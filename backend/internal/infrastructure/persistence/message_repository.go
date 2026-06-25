package persistence

import (
	"fleamarket-backend/internal/domain"

	"gorm.io/gorm"
)

type MessageRepository struct {
	db *gorm.DB
}

func NewMessageRepository(db *gorm.DB) *MessageRepository {
	return &MessageRepository{db: db}
}

func (r *MessageRepository) Create(m *domain.Message) error {
	return r.db.Omit("Product", "Sender").Create(m).Error
}

// ListByProduct は商品IDに紐づくメッセージを古い順に返す。
// callerID が送受信どちらかに含まれるスレッドのみ返す。
func (r *MessageRepository) ListByProduct(productID, callerID uint) ([]domain.Message, error) {
	var msgs []domain.Message

	// 商品の出品者IDを取得
	var product domain.Product
	if err := r.db.Select("seller_id").First(&product, productID).Error; err != nil {
		return nil, err
	}
	sellerID := product.SellerID

	// 出品者 or 購入希望者 (caller) のどちらかであれば閲覧可
	if callerID != sellerID {
		// 購入希望者として参加しているメッセージのみ
		err := r.db.
			Preload("Sender").
			Where("product_id = ? AND (sender_id = ? OR sender_id = ?)", productID, callerID, sellerID).
			Order("created_at ASC").
			Find(&msgs).Error
		return msgs, err
	}

	// 出品者は全メッセージを閲覧可
	err := r.db.
		Preload("Sender").
		Where("product_id = ?", productID).
		Order("created_at ASC").
		Find(&msgs).Error
	return msgs, err
}

func (r *MessageRepository) MarkRead(productID, readerID uint) error {
	return r.db.Model(&domain.Message{}).
		Where("product_id = ? AND sender_id != ? AND is_read = false", productID, readerID).
		Update("is_read", true).Error
}
