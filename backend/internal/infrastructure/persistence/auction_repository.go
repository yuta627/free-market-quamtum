package persistence

import (
	"errors"
	"time"

	"fleamarket-backend/internal/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type AuctionRepository struct {
	db *gorm.DB
}

func NewAuctionRepository(db *gorm.DB) *AuctionRepository {
	return &AuctionRepository{db: db}
}

func (r *AuctionRepository) Create(a *domain.Auction) error {
	return r.db.Create(a).Error
}

// CreateWithProduct は商品とオークションを1トランザクションで作成する。
// 片方だけコミットされる孤立行を防ぐ。
func (r *AuctionRepository) CreateWithProduct(product *domain.Product, auction *domain.Auction) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(product).Error; err != nil {
			return err
		}
		auction.ProductID = product.ID
		return tx.Create(auction).Error
	})
}

func (r *AuctionRepository) FindByID(id uint) (*domain.Auction, error) {
	var a domain.Auction
	err := r.db.Preload("Product.Seller").Preload("Bids.Bidder").First(&a, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &a, err
}

func (r *AuctionRepository) ListActive(limit, offset int) ([]domain.Auction, int64, error) {
	var auctions []domain.Auction
	var total int64

	if err := r.db.Model(&domain.Auction{}).
		Where("status = 'active' AND ends_at > NOW()").
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := r.db.Where("status = 'active' AND ends_at > NOW()").
		Preload("Product.Seller").
		Order("ends_at ASC").
		Limit(limit).Offset(offset).
		Find(&auctions).Error
	return auctions, total, err
}

// FindTopBids は最高額の入札をすべて返す（同額タイの場合に複数返る）。
func (r *AuctionRepository) FindTopBids(auctionID uint) ([]domain.Bid, error) {
	var a domain.Auction
	if err := r.db.First(&a, auctionID).Error; err != nil {
		return nil, err
	}
	var bids []domain.Bid
	err := r.db.Where("auction_id = ? AND amount = ?", auctionID, a.CurrentPrice).
		Order("created_at ASC").
		Find(&bids).Error
	return bids, err
}

// Finalize は落札者・落札入札を記録してステータスを ended に更新する。
func (r *AuctionRepository) Finalize(auctionID, winnerID, winnerBidID uint) (*domain.Auction, error) {
	err := r.db.Transaction(func(tx *gorm.DB) error {
		var a domain.Auction
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&a, auctionID).Error; err != nil {
			return err
		}
		if a.Status != "active" {
			return domain.ErrAlreadyFinalized
		}
		return tx.Model(&a).Updates(map[string]interface{}{
			"status":        "ended",
			"winner_id":     winnerID,
			"winner_bid_id": winnerBidID,
		}).Error
	})
	if err != nil {
		return nil, err
	}
	return r.FindByID(auctionID)
}

func (r *AuctionRepository) PlaceBid(bid *domain.Bid, newPrice int) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var a domain.Auction
		// GORM v2: FOR UPDATE 悲観的ロック
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&a, bid.AuctionID).Error; err != nil {
			return err
		}
		if a.Status != "active" || a.EndsAt.Before(time.Now()) {
			return domain.ErrAuctionEnded
		}
		if bid.Amount <= a.CurrentPrice {
			return domain.ErrBidTooLow
		}
		if err := tx.Create(bid).Error; err != nil {
			return err
		}
		return tx.Model(&a).Updates(map[string]interface{}{
			"current_price": newPrice,
			"bid_count":     gorm.Expr("bid_count + 1"),
		}).Error
	})
}
