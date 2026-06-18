package usecase

import (
	"errors"
	"time"

	"fleamarket-backend/internal/domain"
	"fleamarket-backend/internal/infrastructure"
	"fleamarket-backend/internal/infrastructure/persistence"
)

var (
	ErrAuctionNotFound = errors.New("auction not found")
	ErrSelfBid         = errors.New("cannot bid on your own auction")
)

type AuctionUsecase struct {
	auctionRepo *persistence.AuctionRepository
	productRepo *persistence.ProductRepository
	qrng        *infrastructure.QRNGClient
}

func NewAuctionUsecase(ar *persistence.AuctionRepository, pr *persistence.ProductRepository, qrng *infrastructure.QRNGClient) *AuctionUsecase {
	return &AuctionUsecase{auctionRepo: ar, productRepo: pr, qrng: qrng}
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

var ErrNotSeller = errors.New("only the seller can finalize the auction")

// FinalizeAuction はオークションを終了し、QRNGで同額最高入札者の中から落札者を決定する。
func (u *AuctionUsecase) FinalizeAuction(auctionID, callerID uint) (*domain.Auction, error) {
	a, err := u.auctionRepo.FindByID(auctionID)
	if err != nil {
		return nil, err
	}
	if a == nil {
		return nil, ErrAuctionNotFound
	}
	if a.Product.SellerID != callerID {
		return nil, ErrNotSeller
	}
	if a.Status != "active" {
		return nil, domain.ErrAlreadyFinalized
	}
	if time.Now().Before(a.EndsAt) {
		return nil, domain.ErrAuctionNotEnded
	}

	topBids, err := u.auctionRepo.FindTopBids(auctionID)
	if err != nil {
		return nil, err
	}
	if len(topBids) == 0 {
		return nil, domain.ErrNoBids
	}

	// 同額入札者が複数いる場合はQRNGで公平に抽選
	winnerBid := topBids[0]
	if len(topBids) > 1 {
		result, qErr := u.qrng.GetRandom(0, len(topBids)-1, "auction_tiebreak")
		if qErr == nil {
			winnerBid = topBids[result.Value]
		}
		// QRNG失敗時は最初の入札者（早い者勝ち）
	}

	return u.auctionRepo.Finalize(auctionID, winnerBid.BidderID, winnerBid.ID)
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
