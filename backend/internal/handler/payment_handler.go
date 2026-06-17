package handler

import (
	"errors"
	"net/http"
	"strconv"

	"fleamarket-backend/internal/usecase"

	"github.com/gin-gonic/gin"
)

type PaymentHandler struct {
	uc *usecase.PaymentUsecase
}

func NewPaymentHandler(uc *usecase.PaymentUsecase) *PaymentHandler {
	return &PaymentHandler{uc: uc}
}

func (h *PaymentHandler) CreateCheckout(c *gin.Context) {
	userID, _ := c.Get(ContextKeyUserID)

	productID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	out, err := h.uc.CreateCheckout(uint(productID), userID.(uint))
	switch {
	case errors.Is(err, usecase.ErrProductNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "商品が見つかりません"})
	case errors.Is(err, usecase.ErrCannotBuyOwnProduct):
		c.JSON(http.StatusBadRequest, gin.H{"error": "自分が出品した商品は購入できません"})
	case errors.Is(err, usecase.ErrProductNotAvailable):
		c.JSON(http.StatusConflict, gin.H{"error": "この商品は既に売り切れています"})
	case err != nil:
		_ = c.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "決済の開始に失敗しました"})
	default:
		c.JSON(http.StatusOK, out)
	}
}

type confirmPurchaseRequest struct {
	PaymentIntentID string `json:"payment_intent_id" binding:"required"`
}

func (h *PaymentHandler) ConfirmPurchase(c *gin.Context) {
	userID, _ := c.Get(ContextKeyUserID)

	productID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req confirmPurchaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payment_intent_id is required"})
		return
	}

	p, err := h.uc.ConfirmPurchase(uint(productID), userID.(uint), req.PaymentIntentID)
	switch {
	case errors.Is(err, usecase.ErrProductNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "商品が見つかりません"})
	case errors.Is(err, usecase.ErrPaymentMismatch):
		c.JSON(http.StatusBadRequest, gin.H{"error": "決済情報が一致しません"})
	case errors.Is(err, usecase.ErrPaymentNotSucceeded):
		c.JSON(http.StatusPaymentRequired, gin.H{"error": "決済が完了していません"})
	case errors.Is(err, usecase.ErrProductNotAvailable):
		c.JSON(http.StatusConflict, gin.H{"error": "この商品は既に売り切れています"})
	case err != nil:
		_ = c.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "購入の確定に失敗しました"})
	default:
		c.JSON(http.StatusOK, p)
	}
}
