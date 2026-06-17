package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"fleamarket-backend/internal/domain"
	"fleamarket-backend/internal/usecase"

	"github.com/gin-gonic/gin"
)

type ProductHandler struct {
	productUC *usecase.ProductUsecase
}

func NewProductHandler(uc *usecase.ProductUsecase) *ProductHandler {
	return &ProductHandler{productUC: uc}
}

type createProductRequest struct {
	Title       string `json:"title"       binding:"required,min=1,max=200"`
	Description string `json:"description"`
	Price       int    `json:"price"       binding:"required,min=0"`
	Condition   string `json:"condition"   binding:"required,oneof=new like_new good fair poor"`
	ImageURLs   []string `json:"image_urls"`
}

func (h *ProductHandler) Create(c *gin.Context) {
	userID, _ := c.Get(ContextKeyUserID)

	var req createProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	imageJSON := "[]"
	if len(req.ImageURLs) > 0 {
		b, _ := json.Marshal(req.ImageURLs)
		imageJSON = string(b)
	}

	p, err := h.productUC.Create(usecase.CreateProductInput{
		SellerID:    userID.(uint),
		Title:       req.Title,
		Description: req.Description,
		Price:       req.Price,
		Condition:   domain.ProductCondition(req.Condition),
		ImageURLs:   imageJSON,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "出品に失敗しました"})
		return
	}
	c.JSON(http.StatusCreated, p)
}

func (h *ProductHandler) List(c *gin.Context) {
	status := c.DefaultQuery("status", "on_sale")
	query := c.DefaultQuery("q", "")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	out, err := h.productUC.List(usecase.ListProductsInput{
		Status: status,
		Query:  query,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "一覧取得に失敗しました"})
		return
	}
	c.JSON(http.StatusOK, out)
}

func (h *ProductHandler) GetByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	p, err := h.productUC.GetByID(uint(id))
	if errors.Is(err, usecase.ErrProductNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "商品が見つかりません"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "取得に失敗しました"})
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *ProductHandler) ListMine(c *gin.Context) {
	userID, _ := c.Get(ContextKeyUserID)

	products, err := h.productUC.ListMine(userID.(uint))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "出品商品の取得に失敗しました"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"products": products})
}

func (h *ProductHandler) ListPurchased(c *gin.Context) {
	userID, _ := c.Get(ContextKeyUserID)

	products, err := h.productUC.ListPurchased(userID.(uint))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "購入商品の取得に失敗しました"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"products": products})
}

func (h *ProductHandler) Purchase(c *gin.Context) {
	userID, _ := c.Get(ContextKeyUserID)

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	p, err := h.productUC.Purchase(uint(id), userID.(uint))
	switch {
	case errors.Is(err, usecase.ErrProductNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "商品が見つかりません"})
	case errors.Is(err, usecase.ErrCannotBuyOwnProduct):
		c.JSON(http.StatusBadRequest, gin.H{"error": "自分が出品した商品は購入できません"})
	case errors.Is(err, usecase.ErrProductNotAvailable):
		c.JSON(http.StatusConflict, gin.H{"error": "この商品は既に売り切れています"})
	case err != nil:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "購入処理に失敗しました"})
	default:
		c.JSON(http.StatusOK, p)
	}
}
