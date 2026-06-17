package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"fleamarket-backend/internal/domain"
	"fleamarket-backend/internal/usecase"

	"github.com/gin-gonic/gin"
)

type AuctionHandler struct {
	uc *usecase.AuctionUsecase
}

func NewAuctionHandler(uc *usecase.AuctionUsecase) *AuctionHandler {
	return &AuctionHandler{uc: uc}
}

type createAuctionRequest struct {
	Title         string   `json:"title"          binding:"required,min=1,max=200"`
	Description   string   `json:"description"`
	Condition     string   `json:"condition"      binding:"required,oneof=new like_new good fair poor"`
	ImageURLs     []string `json:"image_urls"`
	StartingPrice int      `json:"starting_price" binding:"min=0,max=100000000"`
	EndsAt        string   `json:"ends_at"        binding:"required"`
}

func (h *AuctionHandler) Create(c *gin.Context) {
	userID, _ := c.Get(ContextKeyUserID)

	var req createAuctionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	endsAt, err := time.Parse(time.RFC3339, req.EndsAt)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ends_at must be RFC3339 format"})
		return
	}
	if endsAt.Before(time.Now().Add(time.Hour)) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ends_at must be at least 1 hour from now"})
		return
	}

	imageJSON := "[]"
	if len(req.ImageURLs) > 0 {
		b, _ := json.Marshal(req.ImageURLs)
		imageJSON = string(b)
	}

	a, err := h.uc.Create(usecase.CreateAuctionInput{
		SellerID:      userID.(uint),
		Title:         req.Title,
		Description:   req.Description,
		Condition:     domain.ProductCondition(req.Condition),
		ImageURLs:     imageJSON,
		StartingPrice: req.StartingPrice,
		EndsAt:        endsAt,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "オークション出品に失敗しました"})
		return
	}
	c.JSON(http.StatusCreated, a)
}

func (h *AuctionHandler) List(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	auctions, total, err := h.uc.List(limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "一覧取得に失敗しました"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"auctions": auctions, "total": total})
}

func (h *AuctionHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	a, err := h.uc.GetByID(uint(id))
	if errors.Is(err, usecase.ErrAuctionNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "オークションが見つかりません"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "取得に失敗しました"})
		return
	}
	c.JSON(http.StatusOK, a)
}

func (h *AuctionHandler) PlaceBid(c *gin.Context) {
	userID, _ := c.Get(ContextKeyUserID)

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req struct {
		Amount int `json:"amount" binding:"required,min=1,max=100000000"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	a, err := h.uc.PlaceBid(uint(id), userID.(uint), req.Amount)
	switch {
	case errors.Is(err, usecase.ErrAuctionNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "オークションが見つかりません"})
	case errors.Is(err, usecase.ErrSelfBid):
		c.JSON(http.StatusForbidden, gin.H{"error": "自分が出品したオークションには入札できません"})
	case errors.Is(err, domain.ErrAuctionEnded):
		c.JSON(http.StatusConflict, gin.H{"error": "このオークションは終了しています"})
	case errors.Is(err, domain.ErrBidTooLow):
		c.JSON(http.StatusBadRequest, gin.H{"error": "現在価格より高い金額で入札してください"})
	case err != nil:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "入札に失敗しました"})
	default:
		c.JSON(http.StatusOK, a)
	}
}
