package persistence

import (
	"fleamarket-backend/internal/domain"

	"gorm.io/gorm"
)

type NotificationRepository struct {
	db *gorm.DB
}

func NewNotificationRepository(db *gorm.DB) *NotificationRepository {
	db.AutoMigrate(&domain.Notification{})
	return &NotificationRepository{db: db}
}

func (r *NotificationRepository) Create(n *domain.Notification) error {
	return r.db.Create(n).Error
}

func (r *NotificationRepository) ListByUserID(userID uint) ([]domain.Notification, error) {
	var ns []domain.Notification
	err := r.db.Where("user_id = ?", userID).Order("created_at DESC").Limit(50).Find(&ns).Error
	return ns, err
}

func (r *NotificationRepository) MarkRead(id, userID uint) error {
	return r.db.Model(&domain.Notification{}).
		Where("id = ? AND user_id = ?", id, userID).
		Update("is_read", true).Error
}

func (r *NotificationRepository) MarkAllRead(userID uint) error {
	return r.db.Model(&domain.Notification{}).
		Where("user_id = ? AND is_read = false", userID).
		Update("is_read", true).Error
}

func (r *NotificationRepository) UnreadCount(userID uint) (int64, error) {
	var count int64
	err := r.db.Model(&domain.Notification{}).
		Where("user_id = ? AND is_read = false", userID).
		Count(&count).Error
	return count, err
}
