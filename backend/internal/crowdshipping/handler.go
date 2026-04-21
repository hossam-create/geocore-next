package crowdshipping

import (
	"log/slog"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/geocore-next/backend/internal/notifications"
	"github.com/geocore-next/backend/pkg/metrics"
	"github.com/geocore-next/backend/pkg/reputation"
	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Handler struct {
	db       *gorm.DB
	notifSvc *notifications.Service
}

type cachedTravelerMatches struct {
	Results []MatchResult
	Expiry  time.Time
}

var (
	travelerMatchesCacheMu sync.RWMutex
	travelerMatchesCache   = map[string]cachedTravelerMatches{}
)

func NewHandler(db *gorm.DB, notifSvc *notifications.Service) *Handler {
	return &Handler{db: db, notifSvc: notifSvc}
}

// ══════════════════════════════════════════════════════════════════════════════
// Trip endpoints
// ══════════════════════════════════════════════════════════════════════════════

type CreateTripReq struct {
	OriginCountry   string  `json:"origin_country" binding:"required"`
	OriginCity      string  `json:"origin_city" binding:"required"`
	OriginAddress   string  `json:"origin_address"`
	DestCountry     string  `json:"dest_country" binding:"required"`
	DestCity        string  `json:"dest_city" binding:"required"`
	DestAddress     string  `json:"dest_address"`
	DepartureDate   string  `json:"departure_date" binding:"required"`
	ArrivalDate     string  `json:"arrival_date" binding:"required"`
	AvailableWeight float64 `json:"available_weight"`
	MaxItems        int     `json:"max_items"`
	PricePerKg      float64 `json:"price_per_kg"`
	BasePrice       float64 `json:"base_price"`
	Currency        string  `json:"currency"`
	Notes           string  `json:"notes"`
	Frequency       string  `json:"frequency"`
}

// POST /api/v1/trips
func (h *Handler) CreateTrip(c *gin.Context) {
	var req CreateTripReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	travelerID, err := uuid.Parse(c.GetString("user_id"))
	if err != nil {
		response.Unauthorized(c)
		return
	}

	dep, err := time.Parse(time.RFC3339, req.DepartureDate)
	if err != nil {
		response.BadRequest(c, "invalid departure_date (RFC3339)")
		return
	}
	arr, err := time.Parse(time.RFC3339, req.ArrivalDate)
	if err != nil {
		response.BadRequest(c, "invalid arrival_date (RFC3339)")
		return
	}

	currency := req.Currency
	if currency == "" {
		currency = "AED"
	}
	freq := req.Frequency
	if freq == "" {
		freq = "one-time"
	}
	maxItems := req.MaxItems
	if maxItems <= 0 {
		maxItems = 5
	}

	trip := Trip{
		TravelerID:      travelerID,
		OriginCountry:   req.OriginCountry,
		OriginCity:      req.OriginCity,
		OriginAddress:   req.OriginAddress,
		DestCountry:     req.DestCountry,
		DestCity:        req.DestCity,
		DestAddress:     req.DestAddress,
		DepartureDate:   dep,
		ArrivalDate:     arr,
		AvailableWeight: req.AvailableWeight,
		MaxItems:        maxItems,
		PricePerKg:      req.PricePerKg,
		BasePrice:       req.BasePrice,
		Currency:        currency,
		Notes:           req.Notes,
		Frequency:       freq,
		Status:          TripStatusActive,
	}

	if err := h.db.Create(&trip).Error; err != nil {
		slog.Error("crowdshipping: create trip failed", "error", err.Error())
		response.InternalError(c, err)
		return
	}
	response.Created(c, trip)
}

// GET /api/v1/trips
func (h *Handler) ListTrips(c *gin.Context) {
	var trips []Trip
	q := h.db.Order("departure_date ASC").Limit(50)

	if status := c.Query("status"); status != "" {
		q = q.Where("status = ?", status)
	} else {
		q = q.Where("status = ?", TripStatusActive)
	}
	if origin := c.Query("origin_country"); origin != "" {
		q = q.Where("origin_country = ?", origin)
	}
	if dest := c.Query("dest_country"); dest != "" {
		q = q.Where("dest_country = ?", dest)
	}
	if mine := c.Query("mine"); mine == "true" {
		q = q.Where("traveler_id = ?", c.GetString("user_id"))
	}

	if err := q.Find(&trips).Error; err != nil {
		response.InternalError(c, err)
		return
	}
	response.OK(c, trips)
}

// GET /api/v1/trips/:id
func (h *Handler) GetTrip(c *gin.Context) {
	var trip Trip
	if err := h.db.Where("id = ?", c.Param("id")).First(&trip).Error; err != nil {
		response.NotFound(c, "trip")
		return
	}
	response.OK(c, trip)
}

// DELETE /api/v1/trips/:id — cancel own trip
func (h *Handler) CancelTrip(c *gin.Context) {
	var trip Trip
	if err := h.db.Where("id = ? AND traveler_id = ?", c.Param("id"), c.GetString("user_id")).First(&trip).Error; err != nil {
		response.NotFound(c, "trip")
		return
	}
	if trip.Status != TripStatusActive {
		response.BadRequest(c, "can only cancel active trips")
		return
	}
	h.db.Model(&trip).Update("status", TripStatusCancelled)
	response.OK(c, gin.H{"message": "trip cancelled"})
}

// ══════════════════════════════════════════════════════════════════════════════
// Delivery Request endpoints
// ══════════════════════════════════════════════════════════════════════════════

type CreateDeliveryReq struct {
	ItemName        string   `json:"item_name" binding:"required"`
	ItemDescription string   `json:"item_description"`
	ItemURL         string   `json:"item_url"`
	ItemPrice       float64  `json:"item_price" binding:"required,gt=0"`
	ItemWeight      *float64 `json:"item_weight"`
	PickupCountry   string   `json:"pickup_country" binding:"required"`
	PickupCity      string   `json:"pickup_city" binding:"required"`
	DeliveryCountry string   `json:"delivery_country" binding:"required"`
	DeliveryCity    string   `json:"delivery_city" binding:"required"`
	Reward          float64  `json:"reward" binding:"required,gt=0"`
	Currency        string   `json:"currency"`
	DeliveryType    string   `json:"delivery_type"`
	Deadline        string   `json:"deadline"`
	Notes           string   `json:"notes"`
}

// POST /api/v1/delivery-requests
func (h *Handler) CreateDeliveryRequest(c *gin.Context) {
	var req CreateDeliveryReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	buyerID, err := uuid.Parse(c.GetString("user_id"))
	if err != nil {
		response.Unauthorized(c)
		return
	}

	currency := req.Currency
	if currency == "" {
		currency = "AED"
	}
	deliveryType := req.DeliveryType
	if deliveryType == "" {
		deliveryType = DeliveryTypeCrowdshipping
	}

	dr := DeliveryRequest{
		BuyerID:         buyerID,
		ItemName:        req.ItemName,
		ItemDescription: req.ItemDescription,
		ItemURL:         req.ItemURL,
		ItemPrice:       req.ItemPrice,
		ItemWeight:      req.ItemWeight,
		PickupCountry:   req.PickupCountry,
		PickupCity:      req.PickupCity,
		DeliveryCountry: req.DeliveryCountry,
		DeliveryCity:    req.DeliveryCity,
		Reward:          req.Reward,
		Currency:        currency,
		DeliveryType:    deliveryType,
		Notes:           req.Notes,
		Status:          DeliveryPending,
	}

	if req.Deadline != "" {
		t, err := time.Parse(time.RFC3339, req.Deadline)
		if err == nil {
			dr.Deadline = &t
		}
	}

	if err := h.db.Create(&dr).Error; err != nil {
		slog.Error("crowdshipping: create delivery request failed", "error", err.Error())
		response.InternalError(c, err)
		return
	}

	// Trigger liquidity engine for crowdshipping requests
	if deliveryType == DeliveryTypeCrowdshipping {
		go TriggerLiquidityEngine(h.db, h.notifSvc, dr.ID)
	}

	response.Created(c, dr)
}

// GET /api/v1/delivery-requests
func (h *Handler) ListDeliveryRequests(c *gin.Context) {
	var requests []DeliveryRequest
	q := h.db.Order("created_at DESC").Limit(50)

	if status := c.Query("status"); status != "" {
		q = q.Where("status = ?", status)
	}
	if origin := c.Query("pickup_country"); origin != "" {
		q = q.Where("pickup_country = ?", origin)
	}
	if dest := c.Query("delivery_country"); dest != "" {
		q = q.Where("delivery_country = ?", dest)
	}
	if mine := c.Query("mine"); mine == "true" {
		q = q.Where("buyer_id = ?", c.GetString("user_id"))
	}

	if err := q.Find(&requests).Error; err != nil {
		response.InternalError(c, err)
		return
	}
	response.OK(c, requests)
}

// GET /api/v1/delivery-requests/:id
func (h *Handler) GetDeliveryRequest(c *gin.Context) {
	var dr DeliveryRequest
	if err := h.db.Where("id = ?", c.Param("id")).First(&dr).Error; err != nil {
		response.NotFound(c, "delivery request")
		return
	}
	response.OK(c, dr)
}

// ══════════════════════════════════════════════════════════════════════════════
// Matching endpoints (ported from mnbara matching-service)
// ══════════════════════════════════════════════════════════════════════════════

// POST /api/v1/delivery-requests/:id/find-travelers
func (h *Handler) FindTravelers(c *gin.Context) {
	metrics.IncMatchingRequestsTotal()
	var dr DeliveryRequest
	if err := h.db.Where("id = ? AND status = ?", c.Param("id"), DeliveryPending).First(&dr).Error; err != nil {
		response.NotFound(c, "pending delivery request")
		return
	}
	if dr.DeliveryType != "" && dr.DeliveryType != DeliveryTypeCrowdshipping {
		response.OK(c, gin.H{
			"delivery_request": dr,
			"matches":          []MatchResult{},
			"skipped":          true,
			"reason":           "matching disabled for non-crowdshipping delivery type",
		})
		return
	}

	cacheKey := "find_travelers:" + dr.ID.String()
	results, hit := getCachedTravelerMatches(cacheKey)
	if !hit {
		var trips []Trip
		q := h.db.Where("status = ? AND origin_country = ? AND dest_country = ? AND departure_date > ?",
			TripStatusActive, dr.PickupCountry, dr.DeliveryCountry, time.Now())

		if dr.ItemWeight != nil && *dr.ItemWeight > 0 {
			q = q.Where("available_weight >= ?", *dr.ItemWeight)
		}

		if err := q.Order("departure_date ASC").Limit(50).Find(&trips).Error; err != nil {
			response.InternalError(c, err)
			return
		}

		results = make([]MatchResult, 0, len(trips))
		for _, trip := range trips {
			// Fetch traveler reputation to boost reliable travelers
			travelerRepScore := 50.0
			if rp, err := reputation.Get(h.db, trip.TravelerID.String()); err == nil {
				travelerRepScore = rp.Score
			}
			score := CalculateMatchScore(&dr, &trip, travelerRepScore)
			var cost float64
			if dr.ItemWeight != nil {
				cost = trip.PricePerKg**dr.ItemWeight + trip.BasePrice
			} else {
				cost = trip.BasePrice
			}
			canDeliver := true
			if dr.ItemWeight != nil && trip.AvailableWeight < *dr.ItemWeight {
				canDeliver = false
			}
			results = append(results, MatchResult{
				Trip:              trip,
				MatchScore:        score,
				TravelerScore:     travelerRepScore,
				EstimatedCost:     cost,
				EstimatedDelivery: trip.ArrivalDate.Format(time.RFC3339),
				CanDeliver:        canDeliver,
			})
		}

		sort.Slice(results, func(i, j int) bool {
			return results[i].MatchScore > results[j].MatchScore
		})
		setCachedTravelerMatches(cacheKey, results, 1*time.Minute)
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))
	pagedResults, page, perPage := paginateMatchResults(results, page, perPage)

	response.OK(c, gin.H{
		"delivery_request": dr,
		"matches":          pagedResults,
		"pagination": gin.H{
			"page":     page,
			"per_page": perPage,
			"total":    len(results),
		},
	})
}

func paginateMatchResults(results []MatchResult, page, perPage int) ([]MatchResult, int, int) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}
	if perPage > 50 {
		perPage = 50
	}

	start := (page - 1) * perPage
	if start > len(results) {
		start = len(results)
	}
	end := start + perPage
	if end > len(results) {
		end = len(results)
	}

	return results[start:end], page, perPage
}

func getCachedTravelerMatches(key string) ([]MatchResult, bool) {
	travelerMatchesCacheMu.RLock()
	entry, ok := travelerMatchesCache[key]
	travelerMatchesCacheMu.RUnlock()
	if !ok || time.Now().After(entry.Expiry) {
		if ok {
			travelerMatchesCacheMu.Lock()
			delete(travelerMatchesCache, key)
			travelerMatchesCacheMu.Unlock()
		}
		return nil, false
	}
	return entry.Results, true
}

func setCachedTravelerMatches(key string, results []MatchResult, ttl time.Duration) {
	travelerMatchesCacheMu.Lock()
	travelerMatchesCache[key] = cachedTravelerMatches{Results: results, Expiry: time.Now().Add(ttl)}
	travelerMatchesCacheMu.Unlock()
}

// GET /api/v1/trips/search
func (h *Handler) SearchTrips(c *gin.Context) {
	var trips []Trip
	q := h.db.Where("status = ?", TripStatusActive)

	if minWeightStr := c.Query("weight_capacity"); minWeightStr != "" {
		if minWeight, err := strconv.ParseFloat(minWeightStr, 64); err == nil {
			q = q.Where("available_weight >= ?", minWeight)
		}
	}
	if maxPriceStr := c.Query("price_per_kg"); maxPriceStr != "" {
		if maxPrice, err := strconv.ParseFloat(maxPriceStr, 64); err == nil {
			q = q.Where("price_per_kg <= ?", maxPrice)
		}
	}
	if depStart := c.Query("departure_start"); depStart != "" {
		if t, err := time.Parse(time.RFC3339, depStart); err == nil {
			q = q.Where("departure_date >= ?", t)
		}
	}
	if depEnd := c.Query("departure_end"); depEnd != "" {
		if t, err := time.Parse(time.RFC3339, depEnd); err == nil {
			q = q.Where("departure_date <= ?", t)
		}
	}

	if err := q.Order("departure_date ASC").Limit(100).Find(&trips).Error; err != nil {
		response.InternalError(c, err)
		return
	}

	type scoredTrip struct {
		Trip
		Score float64 `json:"score"`
	}
	resp := make([]scoredTrip, 0, len(trips))
	reqWeight := 0.0
	if v := c.Query("weight_capacity"); v != "" {
		reqWeight, _ = strconv.ParseFloat(v, 64)
	}
	maxPricePerKg := 0.0
	if v := c.Query("price_per_kg"); v != "" {
		maxPricePerKg, _ = strconv.ParseFloat(v, 64)
	}
	depStart, _ := time.Parse(time.RFC3339, c.Query("departure_start"))
	depEnd, _ := time.Parse(time.RFC3339, c.Query("departure_end"))

	for _, t := range trips {
		weightFit := 1.0
		if reqWeight > 0 {
			weightFit = min(1, t.AvailableWeight/reqWeight)
		}
		priceScore := 1.0
		if maxPricePerKg > 0 {
			priceScore = 1 - min(1, t.PricePerKg/maxPricePerKg)
		}
		timeMatch := 1.0
		if !depStart.IsZero() && !depEnd.IsZero() {
			window := depEnd.Sub(depStart).Hours()
			if window > 0 {
				offset := t.DepartureDate.Sub(depStart).Hours()
				timeMatch = 1 - min(1, max(0, offset/window))
			}
		}
		score := (weightFit * 0.4) + (priceScore * 0.3) + (timeMatch * 0.3)
		resp = append(resp, scoredTrip{Trip: t, Score: score * 100})
	}

	sort.Slice(resp, func(i, j int) bool { return resp[i].Score > resp[j].Score })
	response.OK(c, resp)
}

// POST /api/v1/delivery-requests/:id/match
func (h *Handler) MatchRequest(c *gin.Context) {
	var body struct {
		TripID string `json:"trip_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	buyerID := c.GetString("user_id")

	err := h.db.Transaction(func(tx *gorm.DB) error {
		var dr DeliveryRequest
		if err := tx.Where("id = ? AND buyer_id = ? AND status = ?", c.Param("id"), buyerID, DeliveryPending).First(&dr).Error; err != nil {
			return err
		}

		var trip Trip
		if err := tx.Where("id = ? AND status = ?", body.TripID, TripStatusActive).First(&trip).Error; err != nil {
			return err
		}

		if dr.ItemWeight != nil && trip.AvailableWeight < *dr.ItemWeight {
			return gorm.ErrInvalidData
		}

		score := CalculateMatchScore(&dr, &trip)

		tripID := trip.ID
		travelerID := trip.TravelerID
		tx.Model(&dr).Updates(map[string]any{
			"trip_id":     &tripID,
			"traveler_id": &travelerID,
			"status":      DeliveryMatched,
			"match_score": score,
		})

		if dr.ItemWeight != nil {
			tx.Model(&trip).Update("available_weight", gorm.Expr("available_weight - ?", *dr.ItemWeight))
		}
		tx.Model(&trip).Update("status", TripStatusMatched)

		return nil
	})

	if err != nil {
		response.BadRequest(c, "matching failed: "+err.Error())
		return
	}

	response.OK(c, gin.H{"message": "match requested successfully"})
}

// POST /api/v1/delivery-requests/:id/accept — traveler accepts
func (h *Handler) AcceptMatch(c *gin.Context) {
	travelerID := c.GetString("user_id")

	var dr DeliveryRequest
	if err := h.db.Where("id = ? AND traveler_id = ? AND status = ?", c.Param("id"), travelerID, DeliveryMatched).First(&dr).Error; err != nil {
		response.NotFound(c, "matched delivery request")
		return
	}

	h.db.Model(&dr).Update("status", DeliveryAccepted)
	response.OK(c, gin.H{"message": "match accepted"})
}

// POST /api/v1/delivery-requests/:id/reject — traveler rejects
func (h *Handler) RejectMatch(c *gin.Context) {
	travelerID := c.GetString("user_id")

	err := h.db.Transaction(func(tx *gorm.DB) error {
		var dr DeliveryRequest
		if err := tx.Where("id = ? AND traveler_id = ? AND status = ?", c.Param("id"), travelerID, DeliveryMatched).First(&dr).Error; err != nil {
			return err
		}

		if dr.TripID != nil && dr.ItemWeight != nil {
			tx.Model(&Trip{}).Where("id = ?", *dr.TripID).Updates(map[string]any{
				"status":           TripStatusActive,
				"available_weight": gorm.Expr("available_weight + ?", *dr.ItemWeight),
			})
		} else if dr.TripID != nil {
			tx.Model(&Trip{}).Where("id = ?", *dr.TripID).Update("status", TripStatusActive)
		}

		tx.Model(&dr).Updates(map[string]any{
			"status":      DeliveryPending,
			"trip_id":     nil,
			"traveler_id": nil,
			"match_score": nil,
		})
		return nil
	})

	if err != nil {
		response.BadRequest(c, "reject failed")
		return
	}
	response.OK(c, gin.H{"message": "match rejected"})
}

// POST /api/v1/delivery-requests/:id/confirm-delivery
func (h *Handler) ConfirmDelivery(c *gin.Context) {
	var body struct {
		ProofImageURL string `json:"proof_image_url"`
	}
	c.ShouldBindJSON(&body)

	travelerID := c.GetString("user_id")

	var dr DeliveryRequest
	if err := h.db.Where("id = ? AND traveler_id = ? AND status IN ?", c.Param("id"), travelerID,
		[]DeliveryStatus{DeliveryAccepted, DeliveryPickedUp, DeliveryInTransit}).First(&dr).Error; err != nil {
		response.NotFound(c, "delivery request")
		return
	}

	updates := map[string]any{"status": DeliveryDelivered}
	if body.ProofImageURL != "" {
		updates["proof_image_url"] = body.ProofImageURL
	}
	h.db.Model(&dr).Updates(updates)

	if dr.TripID != nil {
		h.db.Model(&Trip{}).Where("id = ?", *dr.TripID).Update("status", TripStatusCompleted)
	}

	response.OK(c, gin.H{"message": "delivery confirmed"})
}
