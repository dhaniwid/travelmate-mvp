package middleware

import (
	"crypto/subtle"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

// AdminAuthMiddleware ensures the request has the correct X-Admin-Secret header.
// Panics at startup if ADMIN_SECRET is not set.
func AdminAuthMiddleware() gin.HandlerFunc {
	adminSecret := os.Getenv("ADMIN_SECRET")
	if adminSecret == "" {
		log.Fatal("ADMIN_SECRET environment variable must be set. Refusing to start.")
	}

	return func(c *gin.Context) {
		// 1. Check Header
		requestSecret := c.GetHeader("X-Admin-Secret")

		// 2. Constant-time comparison
		if subtle.ConstantTimeCompare([]byte(requestSecret), []byte(adminSecret)) != 1 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Unauthorized: Invalid Admin Secret",
			})
			return
		}

		c.Next()
	}
}
