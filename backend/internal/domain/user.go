package domain

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID             uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	Name           string         `gorm:"type:varchar(100);not null" json:"name"`
	Email          string         `gorm:"type:varchar(255);uniqueIndex;not null" json:"email"`
	HashedPassword string         `gorm:"type:varchar(255);not null" json:"-"`
	AvatarURL      string         `gorm:"type:text" json:"avatar_url"`
	Bio            string         `gorm:"type:text" json:"bio"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`

	Products []Product `gorm:"foreignKey:SellerID" json:"products,omitempty"`
}
