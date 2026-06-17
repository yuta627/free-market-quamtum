package usecase

import (
	"fleamarket-backend/internal/infrastructure"
	"fleamarket-backend/internal/infrastructure/persistence"
)

type RecommendationUsecase struct {
	client      *infrastructure.RecommendationClient
	productRepo *persistence.ProductRepository
}

func NewRecommendationUsecase(client *infrastructure.RecommendationClient, productRepo *persistence.ProductRepository) *RecommendationUsecase {
	return &RecommendationUsecase{client: client, productRepo: productRepo}
}

func (u *RecommendationUsecase) GetQMLSimilarItems(productID uint, limit int) ([]infrastructure.RecommendedItem, error) {
	if limit <= 0 || limit > 50 {
		limit = 10
	}
	return u.client.GetQMLSimilarItems(productID, limit)
}

func (u *RecommendationUsecase) GetSimilarItems(productID uint, limit int) ([]infrastructure.RecommendedItem, error) {
	if limit <= 0 || limit > 50 {
		limit = 10
	}

	product, err := u.productRepo.FindByID(productID)
	if err != nil || product == nil {
		return []infrastructure.RecommendedItem{}, nil
	}

	return u.client.GetSimilarItems(productID, limit, product.Title, product.Price, string(product.Condition))
}
