package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"os"

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
		// If CLERK_PROXY_URL is set (Production with Custom Domain), we use it to allow custom issuer.
		// If empty (Localhost/Dev), we skip it so standard .clerk.accounts.dev tokens are accepted.
		verifyParams := &jwt.VerifyParams{
			Token:  sessionToken,
			Leeway: 5 * time.Second, // Tolerate up to 5s clock skew between client and server
		}

		proxyURL := os.Getenv("CLERK_PROXY_URL")
		if proxyURL != "" {
			fmt.Printf("🌐 [AUTH] Using ProxyURL: %s\n", proxyURL)
			verifyParams.ProxyURL = &proxyURL
		}

		claims, err := jwt.Verify(c.Request.Context(), verifyParams)

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
		// Fallback for different SDK versions
		if userID == "" && claims.ID != "" {
			userID = claims.ID
		}

		// DEBUG LOG (CRITICAL): Print what we found
		fmt.Printf("🔍 AUTH DEBUG: Found Subject='%s' | ID='%s'\n", claims.Subject, claims.ID)

		// 🛑 THE STRICT GATEKEEPER: IF USER ID IS STILL EMPTY -> ABORT!
		if userID == "" {
			fmt.Println("🚨 CRITICAL: Token valid but UserID is EMPTY. Aborting.")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    "unauthorized",
				"message": "Token valid but missing UserID",
			})
			return
		}

		fmt.Printf("✅ AUTH SUCCESS: UserID=%s\n", userID)

		// Set ke context agar bisa dipakai di Handler (misal: tripHandler.ListTrips)
		c.Set("userID", userID)

		// Opsional: Set claims lain jika butuh email/metadata
		// c.Set("user_email", claims.Email)

		c.Next()
	}
}

// OptionalAuthMiddleware attempts to verify the token but does not abort if missing or invalid.
// Used for public endpoints that might need user identity (e.g. GetTrip for registered users).
func OptionalAuthMiddleware(secretKey string) gin.HandlerFunc {
	if secretKey == "" {
		return func(c *gin.Context) { c.Next() } // Fail gracefully
	}

	clerk.SetKey(secretKey)

	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			fmt.Printf("ℹ️ [OPTIONAL-AUTH] No Authorization header found. Proceeding as guest.\n")
			c.Next()
			return
		}

		if !strings.HasPrefix(authHeader, "Bearer ") {
			fmt.Printf("⚠️ [OPTIONAL-AUTH] Invalid header format (No 'Bearer ' prefix). Proceeding as guest.\n")
			c.Next()
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 {
			fmt.Printf("⚠️ [OPTIONAL-AUTH] Invalid header splitting (got %d parts). Proceeding as guest.\n", len(parts))
			c.Next()
			return
		}
		sessionToken := parts[1]

		proxyURL := os.Getenv("CLERK_PROXY_URL")
		verifyParams := &jwt.VerifyParams{
			Token:  sessionToken,
			Leeway: 5 * time.Second, // Tolerate up to 5s clock skew between client and server
		}
		if proxyURL != "" {
			verifyParams.ProxyURL = &proxyURL
		}

		claims, err := jwt.Verify(c.Request.Context(), verifyParams)
		if err != nil {
			fmt.Printf("⚠️ [OPTIONAL-AUTH] Token verification failed: %v. Proceeding as guest.\n", err)
			c.Next()
			return
		}

		userID := claims.Subject
		if userID == "" && claims.ID != "" {
			userID = claims.ID
		}

		if userID != "" {
			fmt.Printf("✅ [OPTIONAL-AUTH] Success! Found UserID=%s\n", userID)
			c.Set("userID", userID)
		} else {
			fmt.Printf("⚠️ [OPTIONAL-AUTH] Token valid but UserID (Subject/ID) is empty.\n")
		}

		c.Next()
	}
}
