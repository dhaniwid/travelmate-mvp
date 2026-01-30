package middleware

import (
	"net/http"
	"os"
	"strings"

	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/clerk/clerk-sdk-go/v2/jwt"
	"github.com/gin-gonic/gin"
)

// AuthMiddleware memverifikasi token JWT dari Clerk
func AuthMiddleware() gin.HandlerFunc {
	// 1. Inisialisasi Kunci Clerk (Hanya sekali saat server start)
	// Pastikan CLERK_SECRET_KEY ada di .env Anda
	secretKey := os.Getenv("CLERK_SECRET_KEY")
	if secretKey == "" {
		// Panic agar developer sadar kalau config belum lengkap
		panic("🔥 FATAL: CLERK_SECRET_KEY is missing in .env")
	}

	// Konfigurasi Clerk SDK dengan Secret Key
	clerk.SetKey(secretKey)

	return func(c *gin.Context) {
		// 2. Ambil Header Authorization
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    "unauthorized",
				"message": "Authorization header is required",
			})
			return
		}

		// 3. Format harus "Bearer <token>"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    "unauthorized",
				"message": "Invalid authorization format. Format: Bearer <token>",
			})
			return
		}

		sessionToken := parts[1]

		// 4. Verifikasi Token ke Clerk
		// SDK ini akan memvalidasi signature & expiration secara otomatis
		claims, err := jwt.Verify(c.Request.Context(), &jwt.VerifyParams{
			Token: sessionToken,
		})

		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    "unauthorized",
				"message": "Invalid or expired token",
				"details": err.Error(),
			})
			return
		}

		// 5. Sukses! Simpan User ID ke Context
		// 'sub' (Subject) di JWT Clerk adalah User ID (contoh: user_2b7...)
		userID := claims.Subject

		// Set ke context agar bisa dipakai di Handler (misal: tripHandler.ListTrips)
		c.Set("user_id", userID)

		// Opsional: Set claims lain jika butuh email/metadata
		// c.Set("user_email", claims.Email)

		c.Next()
	}
}
