package usecase

import (
	"errors"
	"time"

	"fleamarket-backend/internal/domain"
	"fleamarket-backend/internal/infrastructure/persistence"
)

var (
	ErrAuctionNotFound = errors.New("auction not found")
	ErrSelfBid         = errors.New("cannot bid on your own auction")
)

type AuctionUsecase struct {
	auctionRepo *persistence.AuctionRepository
	productRepo *persistence.ProductRepository
}

func NewAuctionUsecase(ar *persistence.AuctionRepository, pr *persistence.ProductRepository) *AuctionUsecase {
	return &AuctionUsecase{auctionRepo: ar, productRepo: pr}
}

type CreateAuctionInput struct {
	SellerID      uint
	Title         string
	Description   string
	Condition     domain.ProductCondition
	ImageURLs     string
	StartingPrice int
	EndsAt        time.Time
}

func (u *AuctionUsecase) Create(in CreateAuctionInput) (*domain.Auction, error) {
	// 商品とオークションを1トランザクションで作成 — 片方だけ残る孤立行を防ぐ
	product := &domain.Product{
		SellerID:    in.SellerID,
		Title:       in.Title,
		Description: in.Description,
		Price:       in.StartingPrice,
		Condition:   in.Condition,
		ImageURLs:   in.ImageURLs,
		Status:      domain.ProductStatusOnSale,
	}
	auction := &domain.Auction{
		StartingPrice: in.StartingPrice,
		CurrentPrice:  in.StartingPrice,
		EndsAt:        in.EndsAt,
		Status:        "active",
	}
	if err := u.auctionRepo.CreateWithProduct(product, auction); err != nil {
		return nil, err
	}
	auction.Product = *product
	return auction, nil
}

func (u *AuctionUsecase) List(limit, offset int) ([]domain.Auction, int64, error) {
	return u.auctionRepo.ListActive(limit, offset)
}

func (u *AuctionUsecase) GetByID(id uint) (*domain.Auction, error) {
	a, err := u.auctionRepo.FindByID(id)
	if err != nil {
		return nil, err
	}
	if a == nil {
		return nil, ErrAuctionNotFound
	}
	return a, nil
}

func (u *AuctionUsecase) PlaceBid(auctionID, bidderID uint, amount int) (*domain.Auction, error) {
	a, err := u.auctionRepo.FindByID(auctionID)
	if err != nil {
		return nil, err
	}
	if a == nil {
		return nil, ErrAuctionNotFound
	}

	// 出品者の自己入札禁止
	if a.Product.SellerID == bidderID {
		return nil, ErrSelfBid
	}

	// 終了時刻チェック（ステータス更新バッチが遅れていても防御）
	if a.EndsAt.Before(time.Now()) {
		return nil, domain.ErrAuctionEnded
	}

	bid := &domain.Bid{
		AuctionID: auctionID,
		BidderID:  bidderID,
		Amount:    amount,
	}
	if err := u.auctionRepo.PlaceBid(bid, amount); err != nil {
		return nil, err
	}
	return u.auctionRepo.FindByID(auctionID)
}
