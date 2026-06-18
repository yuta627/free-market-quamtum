package handler

import (
	"net/http"
	"strconv"

	"fleamarket-backend/internal/infrastructure/persistence"

	"github.com/gin-gonic/gin"
)

type NotificationHandler struct {
	repo *persistence.NotificationRepository
}

func NewNotificationHandler(repo *persistence.NotificationRepository) *NotificationHandler {
	return &NotificationHandler{repo: repo}
}

func (h *NotificationHandler) List(c *gin.Context) {
	userID, _ := c.Get(ContextKeyUserID)
	ns, err := h.repo.ListByUserID(userID.(uint))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "取得に失敗しました"})
		return
	}
	count, _ := h.repo.UnreadCount(userID.(uint))
	c.JSON(http.StatusOK, gin.H{"notifications": ns, "unread_count": count})
}

func (h *NotificationHandler) MarkRead(c *gin.Context) {
	userID, _ := c.Get(ContextKeyUserID)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := h.repo.MarkRead(uint(id), userID.(uint)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新に失敗しました"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *NotificationHandler) MarkAllRead(c *gin.Context) {
	userID, _ := c.Get(ContextKeyUserID)
	if err := h.repo.MarkAllRead(userID.(uint)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新に失敗しました"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
