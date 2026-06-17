package domain

import (
	"time"

	"gorm.io/gorm"
)

type Message struct {
	ID        uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	ProductID uint           `gorm:"not null;index" json:"product_id"`
	SenderID  uint           `gorm:"not null;index" json:"sender_id"`
	Body      string         `gorm:"type:text;not null" json:"body"`
	IsRead    bool           `gorm:"not null;default:false" json:"is_read"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Product Product `gorm:"foreignKey:ProductID" json:"product,omitempty"`
	Sender  User    `gorm:"foreignKey:SenderID" json:"sender,omitempty"`
}
