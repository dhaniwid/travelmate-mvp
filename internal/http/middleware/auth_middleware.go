package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/clerk/clerk-sdk-go/v2/jwks"
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

		// 4. Verifikasi Token ke Clerk
		// CRITICAL: For Custom Domain support in Production, we must point to the custom JWKS endpoint
		jwksClient := &jwks.Client{
			Backend: clerk.NewBackend(&clerk.BackendConfig{
				URL: clerk.String("https://clerk.miru.travel/v1"),
			}),
		}

		proxyURL := "https://clerk.miru.travel"

		claims, err := jwt.Verify(c.Request.Context(), &jwt.VerifyParams{
			Token:      sessionToken,
			JWKSClient: jwksClient, // Custom Domain Fix
			ProxyURL:   &proxyURL,  // Ensure issuer validation passes for custom domain
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
