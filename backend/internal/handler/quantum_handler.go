package handler

import (
	"net/http"
	"strconv"

	"fleamarket-backend/internal/infrastructure"

	"github.com/gin-gonic/gin"
)

type QuantumHandler struct {
	client *infrastructure.QRNGClient
}

func NewQuantumHandler(client *infrastructure.QRNGClient) *QuantumHandler {
	return &QuantumHandler{client: client}
}

func (h *QuantumHandler) GetRandom(c *gin.Context) {
	low, _ := strconv.Atoi(c.DefaultQuery("low", "0"))
	high, _ := strconv.Atoi(c.DefaultQuery("high", "100"))
	purpose := c.DefaultQuery("purpose", "")

	if high <= low {
		c.JSON(http.StatusBadRequest, gin.H{"error": "high must be greater than low"})
		return
	}

	result, err := h.client.GetRandom(low, high, purpose)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}
