package handler

import (
	"net/http"

	"fleamarket-backend/internal/infrastructure/persistence"

	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	userRepo *persistence.UserRepository
}

func NewUserHandler(userRepo *persistence.UserRepository) *UserHandler {
	return &UserHandler{userRepo: userRepo}
}

type updateAddressRequest struct {
	PostalCode  string `json:"postal_code"  binding:"required"`
	Prefecture  string `json:"prefecture"   binding:"required"`
	City        string `json:"city"         binding:"required"`
	AddressLine string `json:"address_line" binding:"required"`
	Building    string `json:"building"`
}

func (h *UserHandler) UpdateAddress(c *gin.Context) {
	userID, _ := c.Get(ContextKeyUserID)

	var req updateAddressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.userRepo.UpdateAddress(userID.(uint), req.PostalCode, req.Prefecture, req.City, req.AddressLine, req.Building); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "住所の更新に失敗しました"})
		return
	}

	user, _ := h.userRepo.FindByID(userID.(uint))
	c.JSON(http.StatusOK, user)
}

func (h *UserHandler) GetMe(c *gin.Context) {
	userID, _ := c.Get(ContextKeyUserID)
	user, err := h.userRepo.FindByID(userID.(uint))
	if err != nil || user == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "ユーザーが見つかりません"})
		return
	}
	c.JSON(http.StatusOK, user)
}
