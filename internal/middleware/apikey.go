package middleware

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

func APIKeyAuth() gin.HandlerFunc {
	requiredKey := os.Getenv("API_KEY")
	return func(c *gin.Context) {
		apikey := c.Query("apikey")
		if apikey == "" {
			apikey = c.GetHeader("X-API-KEY")
		}
		if apikey == "" || (requiredKey != "" && apikey != requiredKey) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "API key requerida o inválida"})
			return
		}
		c.Next()
	}
} 