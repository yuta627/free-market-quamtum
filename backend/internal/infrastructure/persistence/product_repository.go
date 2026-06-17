package persistence

import (
	"errors"

	"fleamarket-backend/internal/domain"

	"gorm.io/gorm"
)

type ProductRepository struct {
	db *gorm.DB
}

func NewProductRepository(db *gorm.DB) *ProductRepository {
	return &ProductRepository{db: db}
}

type ListProductsFilter struct {
	Status string
	Query  string
	Limit  int
	Offset int
}

func (r *ProductRepository) Create(p *domain.Product) error {
	return r.db.Create(p).Error
}


func (r *ProductRepository) FindByID(id uint) (*domain.Product, error) {
	var p domain.Product
	err := r.db.Preload("Seller").First(&p, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &p, err
}

func (r *ProductRepository) List(f ListProductsFilter) ([]domain.Product, int64, error) {
	var products []domain.Product
	var total int64

	q := r.db.Model(&domain.Product{})
	if f.Status != "" {
		q = q.Where("status = ?", f.Status)
	}
	if f.Query != "" {
		q = q.Where("title ILIKE ?", "%"+f.Query+"%")
	}

	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	limit := f.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	err := q.Preload("Seller").
		Order("created_at DESC").
		Limit(limit).
		Offset(f.Offset).
		Find(&products).Error

	return products, total, err
}

func (r *ProductRepository) Update(p *domain.Product) error {
	return r.db.Save(p).Error
}

// ListBySeller returns every product (any status) listed by this seller, newest first.
func (r *ProductRepository) ListBySeller(sellerID uint) ([]domain.Product, error) {
	var products []domain.Product
	err := r.db.Where("seller_id = ?", sellerID).
		Preload("Seller").
		Order("created_at DESC").
		Find(&products).Error
	return products, err
}

// ListByBuyer returns every product this user has purchased, newest first.
func (r *ProductRepository) ListByBuyer(buyerID uint) ([]domain.Product, error) {
	var products []domain.Product
	err := r.db.Where("buyer_id = ?", buyerID).
		Preload("Seller").
		Order("updated_at DESC").
		Find(&products).Error
	return products, err
}

// Purchase atomically marks the product as sold if it is still on sale.
// Returns the number of rows affected (0 means it was already sold/unavailable).
func (r *ProductRepository) Purchase(productID, buyerID uint) (int64, error) {
	result := r.db.Model(&domain.Product{}).
		Where("id = ? AND status = ?", productID, domain.ProductStatusOnSale).
		Updates(map[string]interface{}{
			"status":   domain.ProductStatusSold,
			"buyer_id": buyerID,
		})
	return result.RowsAffected, result.Error
}
