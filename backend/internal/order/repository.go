package order

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/geocore-next/backend/pkg/kafka"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Repository provides data access for orders
type Repository struct {
	db *gorm.DB
}

// NewRepository creates a new order repository
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// DB returns the underlying *gorm.DB (needed for outbox writes outside transactions).
func (r *Repository) DB() *gorm.DB {
	return r.db
}

// Create creates a new order with items
func (r *Repository) Create(ctx context.Context, order *Order) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(order).Error; err != nil {
			return fmt.Errorf("failed to create order: %w", err)
		}
		return nil
	})
}

// CreateWithOutbox creates an order and writes a Kafka outbox entry in the same
// transaction. This is the TRUE outbox pattern — the event is guaranteed to be
// published even if the process crashes after commit.
func (r *Repository) CreateWithOutbox(ctx context.Context, order *Order, requestID string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(order).Error; err != nil {
			return fmt.Errorf("failed to create order: %w", err)
		}

		event := kafka.New(
			"order.created",
			order.ID.String(),
			"order",
			kafka.Actor{Type: "user", ID: order.BuyerID.String()},
			map[string]interface{}{
				"order_id":      order.ID.String(),
				"buyer_id":      order.BuyerID.String(),
				"seller_id":     order.SellerID.String(),
				"amount":        order.Total,
				"currency":      order.Currency,
				"delivery_type": string(order.DeliveryType),
			},
			kafka.EventMeta{TraceID: requestID, Source: "api-service"},
		)
		return kafka.WriteOutbox(tx, kafka.TopicOrders, event)
	})
}

// GetByID retrieves an order by ID with items
func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*Order, error) {
	var order Order
	err := r.db.WithContext(ctx).
		Preload("Items").
		First(&order, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &order, nil
}

// GetByPaymentIntent retrieves an order by Stripe payment intent ID
func (r *Repository) GetByPaymentIntent(ctx context.Context, paymentIntentID string) (*Order, error) {
	var order Order
	err := r.db.WithContext(ctx).
		Preload("Items").
		First(&order, "payment_intent_id = ?", paymentIntentID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &order, nil
}

// ListByBuyer retrieves paginated orders for a buyer
func (r *Repository) ListByBuyer(ctx context.Context, buyerID uuid.UUID, limit, offset int) ([]Order, int64, error) {
	var orders []Order
	var count int64

	// Get total count
	if err := r.db.WithContext(ctx).
		Model(&Order{}).
		Where("buyer_id = ?", buyerID).
		Count(&count).Error; err != nil {
		return nil, 0, err
	}

	// Get orders
	err := r.db.WithContext(ctx).
		Preload("Items").
		Where("buyer_id = ?", buyerID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&orders).Error
	if err != nil {
		return nil, 0, err
	}

	return orders, count, nil
}

// ListBySeller retrieves paginated orders for a seller
func (r *Repository) ListBySeller(ctx context.Context, sellerID uuid.UUID, limit, offset int) ([]Order, int64, error) {
	var orders []Order
	var count int64

	// Get total count
	if err := r.db.WithContext(ctx).
		Model(&Order{}).
		Where("seller_id = ?", sellerID).
		Count(&count).Error; err != nil {
		return nil, 0, err
	}

	// Get orders
	err := r.db.WithContext(ctx).
		Preload("Items").
		Where("seller_id = ?", sellerID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&orders).Error
	if err != nil {
		return nil, 0, err
	}

	return orders, count, nil
}

// UpdateStatus updates the order status and appends to history
func (r *Repository) UpdateStatus(ctx context.Context, orderID uuid.UUID, newStatus OrderStatus, by string, note string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var order Order
		if err := tx.First(&order, "id = ?", orderID).Error; err != nil {
			return err
		}

		// Append new change to existing history
		order.StatusHistory = append(order.StatusHistory, StatusChange{
			Status: newStatus,
			At:     time.Now(),
			By:     by,
			Note:   note,
		})
		order.Status = newStatus

		// Set timestamp fields based on new status
		now := time.Now()
		switch newStatus {
		case StatusConfirmed:
			order.ConfirmedAt = &now
		case StatusShipped:
			order.ShippedAt = &now
		case StatusDelivered:
			order.DeliveredAt = &now
		case StatusCompleted:
			order.CompletedAt = &now
		case StatusCancelled:
			order.CancelledAt = &now
		}

		return tx.Omit("Items").Save(&order).Error
	})
}

// UpdateShipping updates shipping information
func (r *Repository) UpdateShipping(ctx context.Context, orderID uuid.UUID, trackingNumber, carrier string) error {
	return r.db.WithContext(ctx).
		Model(&Order{}).
		Where("id = ?", orderID).
		Updates(map[string]interface{}{
			"tracking_number": trackingNumber,
			"carrier":         carrier,
			"shipped_at":      time.Now(),
			"status":          StatusShipped,
			"updated_at":      time.Now(),
		}).Error
}

// AddOrderItem adds an item to an existing order
func (r *Repository) AddOrderItem(ctx context.Context, item *OrderItem) error {
	return r.db.WithContext(ctx).Create(item).Error
}

// CanTransition checks if a status transition is valid
func CanTransition(from, to OrderStatus) bool {
	validTransitions := map[OrderStatus][]OrderStatus{
		StatusPending:    {StatusConfirmed, StatusCancelled},
		StatusConfirmed:  {StatusProcessing, StatusCancelled},
		StatusProcessing: {StatusShipped, StatusCancelled},
		StatusShipped:    {StatusDelivered},
		StatusDelivered:  {StatusCompleted, StatusDisputed},
		StatusDisputed:   {StatusCompleted, StatusRefunded},
	}

	allowed, ok := validTransitions[from]
	if !ok {
		return false
	}

	for _, s := range allowed {
		if s == to {
			return true
		}
	}
	return false
}
