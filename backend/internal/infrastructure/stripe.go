package infrastructure

import (
	"fmt"
	"os"

	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/paymentintent"
)

type StripeClient struct {
	secretKey string
}

func NewStripeClient() (*StripeClient, error) {
	key := os.Getenv("STRIPE_SECRET_KEY")
	if key == "" {
		return nil, fmt.Errorf("STRIPE_SECRET_KEY is not set")
	}
	return &StripeClient{secretKey: key}, nil
}

// CreatePaymentIntent creates a PaymentIntent for the given amount (JPY, zero-decimal currency).
// metadata is attached so the purchase can later be verified against the originating product/buyer.
func (s *StripeClient) CreatePaymentIntent(amount int64, productID, buyerID uint) (*stripe.PaymentIntent, error) {
	stripe.Key = s.secretKey

	params := &stripe.PaymentIntentParams{
		Amount:             stripe.Int64(amount),
		Currency:           stripe.String(string(stripe.CurrencyJPY)),
		PaymentMethodTypes: stripe.StringSlice([]string{"card", "paypay"}),
		Metadata: map[string]string{
			"product_id": fmt.Sprintf("%d", productID),
			"buyer_id":   fmt.Sprintf("%d", buyerID),
		},
	}

	return paymentintent.New(params)
}

// GetPaymentIntent retrieves the current state of a PaymentIntent from Stripe.
// Always re-fetch before trusting a client-supplied status — never trust the frontend alone.
func (s *StripeClient) GetPaymentIntent(id string) (*stripe.PaymentIntent, error) {
	stripe.Key = s.secretKey
	return paymentintent.Get(id, nil)
}
