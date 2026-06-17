package usecase

import (
	"errors"
	"fmt"

	"fleamarket-backend/internal/domain"
	"fleamarket-backend/internal/infrastructure/persistence"
)

var ErrMessageForbidden = errors.New("not allowed to send message to own product")

type MessageUsecase struct {
	msgRepo     *persistence.MessageRepository
	productRepo *persistence.ProductRepository
}

func NewMessageUsecase(
	msgRepo *persistence.MessageRepository,
	productRepo *persistence.ProductRepository,
) *MessageUsecase {
	return &MessageUsecase{msgRepo: msgRepo, productRepo: productRepo}
}

type SendMessageInput struct {
	ProductID uint
	SenderID  uint
	Body      string
}

func (u *MessageUsecase) Send(in SendMessageInput) (*domain.Message, error) {
	product, err := u.productRepo.FindByID(in.ProductID)
	if err != nil {
		return nil, fmt.Errorf("finding product: %w", err)
	}
	if product == nil {
		return nil, ErrProductNotFound
	}

	// 出品者は自分の商品にメッセージ送信不可（受け取り専用）
	if product.SellerID == in.SenderID {
		return nil, ErrMessageForbidden
	}

	msg := &domain.Message{
		ProductID: in.ProductID,
		SenderID:  in.SenderID,
		Body:      in.Body,
	}
	if err := u.msgRepo.Create(msg); err != nil {
		return nil, fmt.Errorf("creating message: %w", err)
	}

	// Sender をプリロードして返す
	msg.Sender = domain.User{ID: in.SenderID}
	return msg, nil
}

// SendAsEither は出品者・購入希望者どちらからも送れるバージョン（出品者の返信用）
func (u *MessageUsecase) SendReply(in SendMessageInput) (*domain.Message, error) {
	product, err := u.productRepo.FindByID(in.ProductID)
	if err != nil {
		return nil, fmt.Errorf("finding product: %w", err)
	}
	if product == nil {
		return nil, ErrProductNotFound
	}

	msg := &domain.Message{
		ProductID: in.ProductID,
		SenderID:  in.SenderID,
		Body:      in.Body,
	}
	if err := u.msgRepo.Create(msg); err != nil {
		return nil, fmt.Errorf("creating message: %w", err)
	}
	return msg, nil
}

func (u *MessageUsecase) ListByProduct(productID, callerID uint) ([]domain.Message, error) {
	msgs, err := u.msgRepo.ListByProduct(productID, callerID)
	if err != nil {
		return nil, err
	}
	// 自分宛メッセージを既読に
	_ = u.msgRepo.MarkRead(productID, callerID)
	return msgs, nil
}
