package domain

import "time"

// Like は商品×ユーザーごとに1行だけ存在し、解除時も削除せず Liked=false にする。
// これにより「過去に一度でもいいねした履歴」を保持できる。
type Like struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    uint      `gorm:"not null;uniqueIndex:idx_likes_user_product" json:"user_id"`
	ProductID uint      `gorm:"not null;uniqueIndex:idx_likes_user_product" json:"product_id"`
	Liked     bool      `gorm:"not null;default:true" json:"liked"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Product Product `gorm:"foreignKey:ProductID" json:"product,omitempty"`
}
