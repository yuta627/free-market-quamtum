package usecase

import (
	"errors"
	"fmt"
	"strconv"

	"fleamarket-backend/internal/domain"
	"fleamarket-backend/internal/infrastructure"
	"fleamarket-backend/internal/infrastructure/persistence"
)

var ErrPaymentNotSucceeded = errors.New("payment has not succeeded")
var ErrPaymentMismatch = errors.New("payment does not match this product/buyer")

type PaymentUsecase struct {
	stripe           *infrastructure.StripeClient
	productRepo      *persistence.ProductRepository
	notificationRepo *persistence.NotificationRepository
	userRepo         *persistence.UserRepository
}

func NewPaymentUsecase(stripe *infrastructure.StripeClient, productRepo *persistence.ProductRepository, notificationRepo *persistence.NotificationRepository, userRepo *persistence.UserRepository) *PaymentUsecase {
	return &PaymentUsecase{stripe: stripe, productRepo: productRepo, notificationRepo: notificationRepo, userRepo: userRepo}
}

type CheckoutOutput struct {
	ClientSecret    string `json:"client_secret"`
	PaymentIntentID string `json:"payment_intent_id"`
}

// CreateCheckout starts a Stripe PaymentIntent for the given product so the
// frontend can collect card or PayPay payment details via Stripe Elements.
func (u *PaymentUsecase) CreateCheckout(productID, buyerID uint) (*CheckoutOutput, error) {
	if u.stripe == nil {
		return nil, fmt.Errorf("stripe client is not configured")
	}

	p, err := u.productRepo.FindByID(productID)
	if err != nil {
		return nil, fmt.Errorf("finding product: %w", err)
	}
	if p == nil {
		return nil, ErrProductNotFound
	}
	if p.Status != domain.ProductStatusOnSale {
		return nil, ErrProductNotAvailable
	}
	if p.SellerID == buyerID {
		return nil, ErrCannotBuyOwnProduct
	}

	pi, err := u.stripe.CreatePaymentIntent(int64(p.Price), productID, buyerID)
	if err != nil {
		return nil, fmt.Errorf("creating payment intent: %w", err)
	}

	return &CheckoutOutput{
		ClientSecret:    pi.ClientSecret,
		PaymentIntentID: pi.ID,
	}, nil
}

// ConfirmPurchase re-fetches the PaymentIntent from Stripe (never trusting the
// client-reported status alone), verifies it succeeded and matches this
// product/buyer, then finalizes the purchase.
func (u *PaymentUsecase) ConfirmPurchase(productID, buyerID uint, paymentIntentID string) (*domain.Product, error) {
	if u.stripe == nil {
		return nil, fmt.Errorf("stripe client is not configured")
	}

	pi, err := u.stripe.GetPaymentIntent(paymentIntentID)
	if err != nil {
		return nil, fmt.Errorf("retrieving payment intent: %w", err)
	}

	if pi.Metadata["product_id"] != strconv.FormatUint(uint64(productID), 10) ||
		pi.Metadata["buyer_id"] != strconv.FormatUint(uint64(buyerID), 10) {
		return nil, ErrPaymentMismatch
	}

	if pi.Status != "succeeded" {
		return nil, ErrPaymentNotSucceeded
	}

	p, err := u.productRepo.FindByID(productID)
	if err != nil {
		return nil, fmt.Errorf("finding product: %w", err)
	}
	if p == nil {
		return nil, ErrProductNotFound
	}

	affected, err := u.productRepo.Purchase(productID, buyerID)
	if err != nil {
		return nil, fmt.Errorf("purchasing product: %w", err)
	}
	if affected == 0 {
		return nil, ErrProductNotAvailable
	}

	full, err := u.productRepo.FindByID(productID)
	if err != nil {
		return nil, fmt.Errorf("reloading product: %w", err)
	}

	if u.notificationRepo != nil {
		body := fmt.Sprintf("「%s」が購入されました。¥%d\n\n", full.Title, full.Price)

		buyer, _ := u.userRepo.FindByID(buyerID)
		if buyer != nil && buyer.PostalCode != "" {
			body += fmt.Sprintf("【配送先】\n〒%s %s%s%s", buyer.PostalCode, buyer.Prefecture, buyer.City, buyer.AddressLine)
			if buyer.Building != "" {
				body += " " + buyer.Building
			}
			body += "\n\n【配送手順】\n1. 商品を梱包してください\n2. 配送業者（ヤマト・佐川など）に持ち込むか集荷を依頼してください\n3. 上記住所に発送してください\n4. 追跡番号が発行されたら購入者にメッセージで連絡してください"
		} else {
			body += "※ 購入者がまだ住所を登録していません。メッセージで住所を確認してください。"
		}

		_ = u.notificationRepo.Create(&domain.Notification{
			UserID: full.SellerID,
			Title:  "商品が売れました",
			Body:   body,
		})
	}

	return full, nil
}
