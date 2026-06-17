package usecase

import (
	"errors"
	"fmt"

	"fleamarket-backend/internal/domain"
	"fleamarket-backend/internal/infrastructure/persistence"
)

var ErrProductNotFound = errors.New("product not found")
var ErrForbidden = errors.New("forbidden")
var ErrProductNotAvailable = errors.New("product is not available for purchase")
var ErrCannotBuyOwnProduct = errors.New("cannot purchase your own product")

type ProductUsecase struct {
	productRepo *persistence.ProductRepository
}

func NewProductUsecase(r *persistence.ProductRepository) *ProductUsecase {
	return &ProductUsecase{productRepo: r}
}

type CreateProductInput struct {
	SellerID    uint
	Title       string
	Description string
	Price       int
	Condition   domain.ProductCondition
	ImageURLs   string
}

type ListProductsInput struct {
	Status string
	Query  string
	Limit  int
	Offset int
}

type ListProductsOutput struct {
	Products []domain.Product `json:"products"`
	Total    int64            `json:"total"`
	Limit    int              `json:"limit"`
	Offset   int              `json:"offset"`
}

func (u *ProductUsecase) Create(in CreateProductInput) (*domain.Product, error) {
	p := &domain.Product{
		SellerID:    in.SellerID,
		Title:       in.Title,
		Description: in.Description,
		Price:       in.Price,
		Condition:   in.Condition,
		Status:      domain.ProductStatusOnSale,
		ImageURLs:   in.ImageURLs,
	}
	if err := u.productRepo.Create(p); err != nil {
		return nil, fmt.Errorf("creating product: %w", err)
	}
	// Preload seller for response
	full, err := u.productRepo.FindByID(p.ID)
	if err != nil {
		return p, nil
	}
	return full, nil
}

func (u *ProductUsecase) GetByID(id uint) (*domain.Product, error) {
	p, err := u.productRepo.FindByID(id)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, ErrProductNotFound
	}
	return p, nil
}

func (u *ProductUsecase) Purchase(productID, buyerID uint) (*domain.Product, error) {
	p, err := u.productRepo.FindByID(productID)
	if err != nil {
		return nil, fmt.Errorf("finding product: %w", err)
	}
	if p == nil {
		return nil, ErrProductNotFound
	}
	if p.SellerID == buyerID {
		return nil, ErrCannotBuyOwnProduct
	}

	affected, err := u.productRepo.Purchase(productID, buyerID)
	if err != nil {
		return nil, fmt.Errorf("purchasing product: %w", err)
	}
	if affected == 0 {
		return nil, ErrProductNotAvailable
	}

	full, err := u.productRepo.FindByID(productID)
	if err != nil {
		return nil, fmt.Errorf("reloading product: %w", err)
	}
	return full, nil
}

func (u *ProductUsecase) ListMine(sellerID uint) ([]domain.Product, error) {
	return u.productRepo.ListBySeller(sellerID)
}

func (u *ProductUsecase) ListPurchased(buyerID uint) ([]domain.Product, error) {
	return u.productRepo.ListByBuyer(buyerID)
}

func (u *ProductUsecase) List(in ListProductsInput) (*ListProductsOutput, error) {
	products, total, err := u.productRepo.List(persistence.ListProductsFilter{
		Status: in.Status,
		Query:  in.Query,
		Limit:  in.Limit,
		Offset: in.Offset,
	})
	if err != nil {
		return nil, err
	}
	limit := in.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	return &ListProductsOutput{
		Products: products,
		Total:    total,
		Limit:    limit,
		Offset:   in.Offset,
	}, nil
}
