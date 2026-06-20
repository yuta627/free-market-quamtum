package handler

import (
	"net/http"
	"strconv"

	"fleamarket-backend/internal/usecase"

	"github.com/gin-gonic/gin"
)

type RecommendationHandler struct {
	uc *usecase.RecommendationUsecase
}

func NewRecommendationHandler(uc *usecase.RecommendationUsecase) *RecommendationHandler {
	return &RecommendationHandler{uc: uc}
}

func (h *RecommendationHandler) GetClassicalRecommendations(c *gin.Context) {
	productID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	items, err := h.uc.GetClassicalSimilarItems(uint(productID), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "古典推薦の取得に失敗しました"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *RecommendationHandler) GetQKernelRecommendations(c *gin.Context) {
	productID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	items, err := h.uc.GetQKernelSimilarItems(uint(productID), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "量子カーネル推薦の取得に失敗しました"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *RecommendationHandler) GetQMLRecommendations(c *gin.Context) {
	productID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	items, err := h.uc.GetQMLSimilarItems(uint(productID), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "QML推薦の取得に失敗しました"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *RecommendationHandler) GetRecommendations(c *gin.Context) {
	productID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	items, err := h.uc.GetSimilarItems(uint(productID), limit)
	if err != nil {
		_ = c.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "おすすめ商品の取得に失敗しました"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"items": items})
}
