package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/clerk/clerk-sdk-go/v2/jwt"
	"github.com/gin-gonic/gin"
)

// AuthMiddleware memverifikasi token JWT dari Clerk
func AuthMiddleware(secretKey string) gin.HandlerFunc {
	if secretKey == "" {
		panic("🔥 FATAL: CLERK_SECRET_KEY is missing")
	}

	// Konfigurasi Clerk SDK dengan Secret Key
	clerk.SetKey(secretKey)

	// LOUD DEBUG for production troubleshooting
	keyPreview := "EMPTY"
	if len(secretKey) > 10 {
		keyPreview = secretKey[:8] + "..."
	}
	fmt.Printf("🛡️ [AUTH] Clerk initialized with key starting with: %s\n", keyPreview)

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

		// 4. Verifikasi Token ke Clerk (Stateless/Offline Verification)
		// CRITICAL: We use default JWKS fetching (api.clerk.com) to avoid 404 errors on custom domains.
		// We pass ProxyURL to allow tokens issued by the custom domain.
		proxyURL := "https://clerk.miru.travel"

		claims, err := jwt.Verify(c.Request.Context(), &jwt.VerifyParams{
			Token:    sessionToken,
			ProxyURL: &proxyURL, // Allows custom domain issuer
		})

		if err != nil {
			// 🚨 DEBUG: Print exact error to console for troubleshooting
			tokenPreview := "EMPTY"
			if len(sessionToken) > 10 {
				tokenPreview = sessionToken[:10]
			}
			fmt.Printf("🚨 AUTH ERROR: %v | Token Preview: %s...\n", err, tokenPreview)

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
