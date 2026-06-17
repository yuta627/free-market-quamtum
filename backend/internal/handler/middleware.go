package handler

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const ContextKeyUserID = "userID"

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "認証が必要です"})
			return
		}
		tokenStr := strings.TrimPrefix(header, "Bearer ")

		secret := os.Getenv("JWT_SECRET")
		if secret == "" {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "サーバー設定エラー"})
			return
		}

		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(secret), nil
		})
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "トークンが無効です"})
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "トークンが無効です"})
			return
		}

		sub, ok := claims["sub"].(float64)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "トークンが無効です"})
			return
		}
		c.Set(ContextKeyUserID, uint(sub))
		c.Next()
	}
}
