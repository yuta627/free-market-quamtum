package usecase

import (
	"fmt"

	"fleamarket-backend/internal/infrastructure/persistence"
)

type LikeUsecase struct {
	likeRepo    *persistence.LikeRepository
	productRepo *persistence.ProductRepository
}

func NewLikeUsecase(likeRepo *persistence.LikeRepository, productRepo *persistence.ProductRepository) *LikeUsecase {
	return &LikeUsecase{likeRepo: likeRepo, productRepo: productRepo}
}

// ToggleLike flips the liked state for this user/product pair. The
// underlying row is never deleted, so the history survives unliking.
func (u *LikeUsecase) ToggleLike(userID, productID uint) (bool, error) {
	p, err := u.productRepo.FindByID(productID)
	if err != nil {
		return false, fmt.Errorf("finding product: %w", err)
	}
	if p == nil {
		return false, ErrProductNotFound
	}

	existing, err := u.likeRepo.Find(userID, productID)
	if err != nil {
		return false, fmt.Errorf("finding like: %w", err)
	}

	if existing == nil {
		if err := u.likeRepo.Create(userID, productID, true); err != nil {
			return false, fmt.Errorf("creating like: %w", err)
		}
		return true, nil
	}

	newState := !existing.Liked
	if err := u.likeRepo.SetLiked(existing.ID, newState); err != nil {
		return false, fmt.Errorf("updating like: %w", err)
	}
	return newState, nil
}

type LikeHistoryEntry = persistence.HistoryEntry

func (u *LikeUsecase) ListHistory(userID uint) ([]LikeHistoryEntry, error) {
	return u.likeRepo.ListHistoryByUser(userID)
}
