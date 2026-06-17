package handler

import (
	"errors"
	"net/http"
	"strconv"

	"fleamarket-backend/internal/usecase"

	"github.com/gin-gonic/gin"
)

type LikeHandler struct {
	uc *usecase.LikeUsecase
}

func NewLikeHandler(uc *usecase.LikeUsecase) *LikeHandler {
	return &LikeHandler{uc: uc}
}

func (h *LikeHandler) ToggleLike(c *gin.Context) {
	userID, _ := c.Get(ContextKeyUserID)

	productID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	liked, err := h.uc.ToggleLike(userID.(uint), uint(productID))
	switch {
	case errors.Is(err, usecase.ErrProductNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "商品が見つかりません"})
	case err != nil:
		_ = c.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "いいねの更新に失敗しました"})
	default:
		c.JSON(http.StatusOK, gin.H{"liked": liked})
	}
}

type likeHistoryItem struct {
	Product interface{} `json:"product"`
	Liked   bool        `json:"liked"`
}

func (h *LikeHandler) ListLikes(c *gin.Context) {
	userID, _ := c.Get(ContextKeyUserID)

	entries, err := h.uc.ListHistory(userID.(uint))
	if err != nil {
		_ = c.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "いいね一覧の取得に失敗しました"})
		return
	}

	items := make([]likeHistoryItem, 0, len(entries))
	for _, e := range entries {
		items = append(items, likeHistoryItem{Product: e.Product, Liked: e.Liked})
	}

	c.JSON(http.StatusOK, gin.H{"items": items})
}
