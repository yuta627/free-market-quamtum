package domain

import (
	"time"

	"gorm.io/gorm"
)

type ProductStatus string

const (
	ProductStatusOnSale   ProductStatus = "on_sale"
	ProductStatusSold     ProductStatus = "sold"
	ProductStatusDraft    ProductStatus = "draft"
)

type ProductCondition string

const (
	ConditionNew        ProductCondition = "new"
	ConditionLikeNew    ProductCondition = "like_new"
	ConditionGood       ProductCondition = "good"
	ConditionFair       ProductCondition = "fair"
	ConditionPoor       ProductCondition = "poor"
)

type Product struct {
	ID          uint             `gorm:"primaryKey;autoIncrement" json:"id"`
	SellerID    uint             `gorm:"not null;index" json:"seller_id"`
	BuyerID     *uint            `gorm:"index" json:"buyer_id"`
	Title       string           `gorm:"type:varchar(200);not null" json:"title"`
	Description string           `gorm:"type:text" json:"description"`
	Price       int              `gorm:"not null;check:price >= 0" json:"price"`
	Status      ProductStatus    `gorm:"type:varchar(20);not null;default:'on_sale'" json:"status"`
	Condition   ProductCondition `gorm:"type:varchar(20);not null" json:"condition"`
	CategoryID  *uint            `gorm:"index" json:"category_id"`
	ImageURLs   string           `gorm:"type:text" json:"image_urls"` // JSON配列を文字列で保持
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
	DeletedAt   gorm.DeletedAt   `gorm:"index" json:"-"`

	Seller   User      `gorm:"foreignKey:SellerID" json:"seller,omitempty"`
	Buyer    *User     `gorm:"foreignKey:BuyerID" json:"buyer,omitempty"`
	Messages []Message `gorm:"foreignKey:ProductID" json:"messages,omitempty"`
}
