package payments

  import (
  	"fmt"
  	"log/slog"
  	"os"

  	"github.com/stripe/stripe-go/v79"
  	"github.com/stripe/stripe-go/v79/customer"
  	"github.com/stripe/stripe-go/v79/paymentintent"
  	"github.com/stripe/stripe-go/v79/paymentmethod"
  	"github.com/stripe/stripe-go/v79/refund"
  	"github.com/stripe/stripe-go/v79/webhook"
  )

  // InitStripe loads the Stripe secret key from environment and configures the
  // global Stripe client. Call this once at application startup (in RegisterRoutes).
  func InitStripe() {
  	key := os.Getenv("STRIPE_SECRET_KEY")
  	if key == "" {
  		slog.Warn("STRIPE_SECRET_KEY not set — Stripe payments unavailable in this environment")
  		return
  	}
  	stripe.Key = key
  	slog.Info("✅ Stripe initialised")
  }

  // ════════════════════════════════════════════════════════════════════════════
  // Customer
  // ════════════════════════════════════════════════════════════════════════════

  // createStripeCustomer creates a new Stripe Customer and returns its ID.
  func createStripeCustomer(email, name, phone string) (string, error) {
  	p := &stripe.CustomerParams{
  		Email: stripe.String(email),
  		Name:  stripe.String(name),
  	}
  	if phone != "" {
  		p.Phone = stripe.String(phone)
  	}
  	c, err := customer.New(p)
  	if err != nil {
  		return "", fmt.Errorf("stripe: create customer: %w", err)
  	}
  	return c.ID, nil
  }

  // ════════════════════════════════════════════════════════════════════════════
  // PaymentIntent
  // ════════════════════════════════════════════════════════════════════════════

  // createPaymentIntent creates a Stripe PaymentIntent.
  // amount must be in the major currency unit (e.g. 150.00 AED → 15000 fils internally).
  func createPaymentIntent(
  	amount float64,
  	currency string,
  	stripeCustomerID string,
  	description string,
  	metadata map[string]string,
  ) (*stripe.PaymentIntent, error) {
  	amountSmallest := int64(amount * 100)

  	p := &stripe.PaymentIntentParams{
  		Amount:      stripe.Int64(amountSmallest),
  		Currency:    stripe.String(currency),
  		Description: stripe.String(description),
  		AutomaticPaymentMethods: &stripe.PaymentIntentAutomaticPaymentMethodsParams{
  			Enabled: stripe.Bool(true),
  		},
  	}
  	if stripeCustomerID != "" {
  		p.Customer = stripe.String(stripeCustomerID)
  	}
  	for k, v := range metadata {
  		p.AddMetadata(k, v)
  	}

  	pi, err := paymentintent.New(p)
  	if err != nil {
  		return nil, fmt.Errorf("stripe: create payment intent: %w", err)
  	}
  	return pi, nil
  }

  // retrievePaymentIntent fetches the latest status of a PaymentIntent from Stripe.
  func retrievePaymentIntent(piID string) (*stripe.PaymentIntent, error) {
  	pi, err := paymentintent.Get(piID, nil)
  	if err != nil {
  		return nil, fmt.Errorf("stripe: retrieve payment intent %s: %w", piID, err)
  	}
  	return pi, nil
  }

  // issueRefund creates a full or partial refund for the given PaymentIntent ID.
  // amountFils is nil for a full refund, or a pointer to the amount in smallest
  // currency unit (fils/cents) for a partial refund.
  func issueRefund(piID string, amountFils *int64) (*stripe.Refund, error) {
  	p := &stripe.RefundParams{
  		PaymentIntent: stripe.String(piID),
  		Reason:        stripe.String("requested_by_customer"),
  	}
  	if amountFils != nil {
  		p.Amount = amountFils
  	}
  	r, err := refund.New(p)
  	if err != nil {
  		return nil, fmt.Errorf("stripe: issue refund for %s: %w", piID, err)
  	}
  	return r, nil
  }

  // ════════════════════════════════════════════════════════════════════════════
  // Payment methods
  // ════════════════════════════════════════════════════════════════════════════

  // listPaymentMethods returns all card payment methods attached to a Stripe customer.
  func listPaymentMethods(stripeCustomerID string) ([]*stripe.PaymentMethod, error) {
  	params := &stripe.PaymentMethodListParams{
  		Customer: stripe.String(stripeCustomerID),
  		Type:     stripe.String("card"),
  	}
  	params.Limit = stripe.Int64(20)

  	var methods []*stripe.PaymentMethod
  	iter := paymentmethod.List(params)
  	for iter.Next() {
  		methods = append(methods, iter.PaymentMethod())
  	}
  	if err := iter.Err(); err != nil {
  		return nil, fmt.Errorf("stripe: list payment methods: %w", err)
  	}
  	return methods, nil
  }

  // attachPaymentMethod attaches a payment method to a Stripe customer.
  func attachPaymentMethod(pmID, stripeCustomerID string) (*stripe.PaymentMethod, error) {
  	pm, err := paymentmethod.Attach(pmID, &stripe.PaymentMethodAttachParams{
  		Customer: stripe.String(stripeCustomerID),
  	})
  	if err != nil {
  		return nil, fmt.Errorf("stripe: attach payment method %s: %w", pmID, err)
  	}
  	return pm, nil
  }

  // detachPaymentMethod removes a payment method from its Stripe customer.
  func detachPaymentMethod(pmID string) error {
  	_, err := paymentmethod.Detach(pmID, nil)
  	if err != nil {
  		return fmt.Errorf("stripe: detach payment method %s: %w", pmID, err)
  	}
  	return nil
  }

  // ════════════════════════════════════════════════════════════════════════════
  // Webhook signature verification
  // ════════════════════════════════════════════════════════════════════════════

  // VerifyWebhookSignature validates the Stripe-Signature header and returns
  // the parsed Stripe event. Used by the webhook handler (Task 2.2).
  func VerifyWebhookSignature(payload []byte, sigHeader string) (*stripe.Event, error) {
  	secret := os.Getenv("STRIPE_WEBHOOK_SECRET")
  	if secret == "" {
  		return nil, fmt.Errorf("STRIPE_WEBHOOK_SECRET not configured")
  	}
  	event, err := webhook.ConstructEvent(payload, sigHeader, secret)
  	if err != nil {
  		return nil, fmt.Errorf("stripe: invalid webhook signature: %w", err)
  	}
  	return &event, nil
  }

  // stripeErrMsg returns a user-friendly message from a Stripe error.
  func stripeErrMsg(err error) string {
  	if stripeErr, ok := err.(*stripe.Error); ok {
  		switch stripeErr.Code {
  		case stripe.ErrorCodeCardDeclined:
  			return "Your card was declined. Please use a different payment method."
  		case stripe.ErrorCodeExpiredCard:
  			return "Your card has expired. Please update your card details."
  		case stripe.ErrorCodeIncorrectCVC:
  			return "Incorrect security code (CVC). Please check your card details."
  		case stripe.ErrorCodeInsufficientFunds:
  			return "Insufficient funds. Please use a different payment method."
  		case stripe.ErrorCodeAuthenticationRequired:
  			return "Additional authentication required. Please complete the 3D Secure verification."
  		default:
  			if stripeErr.Msg != "" {
  				return stripeErr.Msg
  			}
  		}
  	}
  	return "Payment processing failed. Please try again or contact support."
  }
  