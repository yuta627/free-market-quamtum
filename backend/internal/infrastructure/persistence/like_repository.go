package persistence

import (
	"errors"

	"fleamarket-backend/internal/domain"

	"gorm.io/gorm"
)

type LikeRepository struct {
	db *gorm.DB
}

func NewLikeRepository(db *gorm.DB) *LikeRepository {
	return &LikeRepository{db: db}
}

func (r *LikeRepository) Find(userID, productID uint) (*domain.Like, error) {
	var like domain.Like
	err := r.db.Where("user_id = ? AND product_id = ?", userID, productID).First(&like).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &like, err
}

func (r *LikeRepository) Create(userID, productID uint, liked bool) error {
	return r.db.Create(&domain.Like{UserID: userID, ProductID: productID, Liked: liked}).Error
}

func (r *LikeRepository) SetLiked(id uint, liked bool) error {
	return r.db.Model(&domain.Like{}).Where("id = ?", id).Update("liked", liked).Error
}

// HistoryEntry pairs a product with whether it is currently liked by the user.
type HistoryEntry struct {
	Product domain.Product
	Liked   bool
}

// ListHistoryByUser returns every product the user has ever liked (including
// ones since unliked), most recently touched first.
func (r *LikeRepository) ListHistoryByUser(userID uint) ([]HistoryEntry, error) {
	var likes []domain.Like
	err := r.db.Where("user_id = ?", userID).
		Preload("Product").
		Preload("Product.Seller").
		Order("updated_at DESC").
		Find(&likes).Error
	if err != nil {
		return nil, err
	}

	entries := make([]HistoryEntry, 0, len(likes))
	for _, l := range likes {
		entries = append(entries, HistoryEntry{Product: l.Product, Liked: l.Liked})
	}
	return entries, nil
}
