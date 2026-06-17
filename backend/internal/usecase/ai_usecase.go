package usecase

import (
	"context"
	"fmt"

	"fleamarket-backend/internal/infrastructure"
	"fleamarket-backend/internal/infrastructure/persistence"
)

type AIUsecase struct {
	gemini      *infrastructure.GeminiClient
	productRepo *persistence.ProductRepository
}

func NewAIUsecase(gemini *infrastructure.GeminiClient, productRepo *persistence.ProductRepository) *AIUsecase {
	return &AIUsecase{gemini: gemini, productRepo: productRepo}
}

func (u *AIUsecase) GenerateDescription(ctx context.Context, title, keywords string) (string, error) {
	if u.gemini == nil {
		return "", fmt.Errorf("gemini client is not configured")
	}
	return u.gemini.GenerateProductDescription(ctx, title, keywords)
}

func (u *AIUsecase) AnswerProductQuestion(ctx context.Context, productID uint, question string) (string, error) {
	if u.gemini == nil {
		return "", fmt.Errorf("gemini client is not configured")
	}

	product, err := u.productRepo.FindByID(productID)
	if err != nil {
		return "", fmt.Errorf("finding product: %w", err)
	}
	if product == nil {
		return "", ErrProductNotFound
	}

	return u.gemini.AnswerProductQuestion(ctx, product.Title, product.Description, question)
}
