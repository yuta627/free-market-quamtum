package handler

import (
	"errors"
	"net/http"
	"strconv"

	"fleamarket-backend/internal/usecase"

	"github.com/gin-gonic/gin"
)

type AIHandler struct {
	uc *usecase.AIUsecase
}

func NewAIHandler(uc *usecase.AIUsecase) *AIHandler {
	return &AIHandler{uc: uc}
}

type generateDescriptionRequest struct {
	Title    string `json:"title" binding:"required"`
	Keywords string `json:"keywords"`
}

func (h *AIHandler) GenerateDescription(c *gin.Context) {
	var req generateDescriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "title is required"})
		return
	}

	desc, err := h.uc.GenerateDescription(c.Request.Context(), req.Title, req.Keywords)
	if err != nil {
		_ = c.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "AIによる生成に失敗しました: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"description": desc})
}

type askQuestionRequest struct {
	Question string `json:"question" binding:"required"`
}

func (h *AIHandler) AskQuestion(c *gin.Context) {
	idStr := c.Param("id")
	productID, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req askQuestionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "question is required"})
		return
	}

	answer, err := h.uc.AnswerProductQuestion(c.Request.Context(), uint(productID), req.Question)
	switch {
	case errors.Is(err, usecase.ErrProductNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "商品が見つかりません"})
	case err != nil:
		_ = c.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "AIによる回答生成に失敗しました"})
	default:
		c.JSON(http.StatusOK, gin.H{"answer": answer})
	}
}
