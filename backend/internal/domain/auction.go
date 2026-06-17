package domain

import "time"

type Auction struct {
	ID            uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	ProductID     uint      `gorm:"not null;index" json:"product_id"`
	StartingPrice int       `gorm:"not null" json:"starting_price"`
	CurrentPrice  int       `gorm:"not null" json:"current_price"`
	BidCount      int       `gorm:"not null;default:0" json:"bid_count"`
	EndsAt        time.Time `gorm:"not null" json:"ends_at"`
	Status        string    `gorm:"type:varchar(20);not null;default:'active'" json:"status"`
	WinnerID      *uint     `gorm:"index" json:"winner_id,omitempty"`
	WinnerBidID   *uint     `gorm:"index" json:"winner_bid_id,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`

	Product Product `gorm:"foreignKey:ProductID" json:"product,omitempty"`
	Bids    []Bid   `gorm:"foreignKey:AuctionID" json:"bids,omitempty"`
	Winner  *User   `gorm:"foreignKey:WinnerID" json:"winner,omitempty"`
}

type Bid struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	AuctionID uint      `gorm:"not null;index" json:"auction_id"`
	BidderID  uint      `gorm:"not null;index" json:"bidder_id"`
	Amount    int       `gorm:"not null" json:"amount"`
	CreatedAt time.Time `json:"created_at"`

	Bidder User `gorm:"foreignKey:BidderID" json:"bidder,omitempty"`
}
