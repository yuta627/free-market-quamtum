package handler

import (
	"errors"
	"net/http"
	"strconv"

	"fleamarket-backend/internal/usecase"

	"github.com/gin-gonic/gin"
)

type MessageHandler struct {
	msgUC *usecase.MessageUsecase
}

func NewMessageHandler(uc *usecase.MessageUsecase) *MessageHandler {
	return &MessageHandler{msgUC: uc}
}

type sendMessageRequest struct {
	Body string `json:"body" binding:"required,min=1,max=2000"`
}

// POST /api/v1/products/:id/messages
func (h *MessageHandler) Send(c *gin.Context) {
	userID := c.MustGet(ContextKeyUserID).(uint)

	productID, err := parseUintParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid product id"})
		return
	}

	var req sendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	msg, err := h.msgUC.SendReply(usecase.SendMessageInput{
		ProductID: productID,
		SenderID:  userID,
		Body:      req.Body,
	})
	if errors.Is(err, usecase.ErrProductNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "商品が見つかりません"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, msg)
}

// GET /api/v1/products/:id/messages
func (h *MessageHandler) List(c *gin.Context) {
	userID := c.MustGet(ContextKeyUserID).(uint)

	productID, err := parseUintParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid product id"})
		return
	}

	msgs, err := h.msgUC.ListByProduct(productID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "取得に失敗しました"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"messages": msgs})
}

func parseUintParam(c *gin.Context, key string) (uint, error) {
	v, err := strconv.ParseUint(c.Param(key), 10, 64)
	return uint(v), err
}
