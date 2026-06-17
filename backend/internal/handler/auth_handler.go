package handler

import (
	"errors"
	"net/http"

	"fleamarket-backend/internal/usecase"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	authUC *usecase.AuthUsecase
}

func NewAuthHandler(authUC *usecase.AuthUsecase) *AuthHandler {
	return &AuthHandler{authUC: authUC}
}

type signupRequest struct {
	Name     string `json:"name"     binding:"required,min=1,max=100"`
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

type loginRequest struct {
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

func (h *AuthHandler) Signup(c *gin.Context) {
	var req signupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	out, err := h.authUC.Signup(usecase.SignupInput{
		Name:     req.Name,
		Email:    req.Email,
		Password: req.Password,
	})
	if errors.Is(err, usecase.ErrEmailAlreadyExists) {
		c.JSON(http.StatusConflict, gin.H{"error": "このメールアドレスはすでに登録されています"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "サーバーエラーが発生しました"})
		return
	}

	c.JSON(http.StatusCreated, out)
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	out, err := h.authUC.Login(usecase.LoginInput{
		Email:    req.Email,
		Password: req.Password,
	})
	if errors.Is(err, usecase.ErrInvalidCredentials) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "メールアドレスまたはパスワードが正しくありません"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "サーバーエラーが発生しました"})
		return
	}

	c.JSON(http.StatusOK, out)
}
