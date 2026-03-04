package middleware

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
	"travelmate/internal/domain"

	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/clerk/clerk-sdk-go/v2/jwt"
	clerkuser "github.com/clerk/clerk-sdk-go/v2/user"
	"github.com/gin-gonic/gin"
)

// UserSyncer is a minimal interface so the middleware can upsert a Clerk user
// into the local users table without importing the full repositories package.
type UserSyncer interface {
	UpsertUser(ctx context.Context, user *domain.User) error
}

// AuthMiddleware verifies Clerk JWT tokens and enriches the Gin context with
// userID, email, and name by fetching the full user profile from the Clerk
// Backend API. On every authenticated request it upserts the user into the
// local users table (INSERT ON CONFLICT) so new signups are captured instantly.
func AuthMiddleware(secretKey string, userRepo UserSyncer) gin.HandlerFunc {
	if secretKey == "" {
		panic("🔥 FATAL: CLERK_SECRET_KEY is missing")
	}

	clerk.SetKey(secretKey)

	keyPreview := "EMPTY"
	if len(secretKey) > 10 {
		keyPreview = secretKey[:8] + "..."
	}
	fmt.Printf("🛡️ [AUTH] Clerk initialized with key starting with: %s\n", keyPreview)

	return func(c *gin.Context) {
		// 1. Get Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    "unauthorized",
				"message": "Authorization header is required",
			})
			return
		}

		// 2. Validate Bearer format
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    "unauthorized",
				"message": "Invalid authorization format. Format: Bearer <token>",
			})
			return
		}
		sessionToken := parts[1]

		// 3. Verify JWT via Clerk SDK (offline/stateless)
		verifyParams := &jwt.VerifyParams{
			Token:  sessionToken,
			Leeway: 5 * time.Second,
		}
		proxyURL := os.Getenv("CLERK_PROXY_URL")
		if proxyURL != "" {
			fmt.Printf("🌐 [AUTH] Using ProxyURL: %s\n", proxyURL)
			verifyParams.ProxyURL = &proxyURL
		}

		claims, err := jwt.Verify(c.Request.Context(), verifyParams)
		if err != nil {
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

		// 4. Extract userID from JWT subject
		userID := claims.Subject
		if userID == "" && claims.ID != "" {
			userID = claims.ID
		}
		//fmt.Printf("🔍 AUTH DEBUG: Found Subject='%s' | ID='%s'\n", claims.Subject, claims.ID)

		if userID == "" {
			fmt.Println("🚨 CRITICAL: Token valid but UserID is EMPTY. Aborting.")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    "unauthorized",
				"message": "Token valid but missing UserID",
			})
			return
		}

		//fmt.Printf("✅ AUTH SUCCESS: UserID=%s\n", userID)
		c.Set("userID", userID)

		// 5. Fetch full user profile from Clerk Backend API to sync email + name.
		//    Non-fatal: if the fetch fails, the request continues without email context.
		clerkUserObj, fetchErr := clerkuser.Get(c.Request.Context(), userID)
		if fetchErr == nil && clerkUserObj != nil {
			// Extract primary email — match by PrimaryEmailAddressID first
			email := ""
			if clerkUserObj.PrimaryEmailAddressID != nil {
				for _, ea := range clerkUserObj.EmailAddresses {
					if ea != nil && ea.ID == *clerkUserObj.PrimaryEmailAddressID {
						email = ea.EmailAddress
						break
					}
				}
			}
			// Fallback: use first available email address
			if email == "" && len(clerkUserObj.EmailAddresses) > 0 && clerkUserObj.EmailAddresses[0] != nil {
				email = clerkUserObj.EmailAddresses[0].EmailAddress
			}

			// Build full name from first + last
			name := ""
			if clerkUserObj.FirstName != nil {
				name = *clerkUserObj.FirstName
			}
			if clerkUserObj.LastName != nil {
				if name != "" {
					name += " "
				}
				name += *clerkUserObj.LastName
			}

			if email != "" {
				c.Set("email", email)
			}
			if name != "" {
				c.Set("name", name)
			}
			//fmt.Printf("📧 AUTH: Synced email=%s name=%s for userID=%s\n", email, name, userID)

			// 6. Upsert user into DB asynchronously (fire-and-forget).
			//    INSERT ON CONFLICT — creates the row for new users, backfills email/name for existing ones.
			if userRepo != nil && email != "" {
				go func(uid, em, nm string) {
					bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					defer cancel()
					if dbErr := userRepo.UpsertUser(bgCtx, &domain.User{
						UserID: uid,
						Email:  em,
						Name:   nm,
					}); dbErr != nil {
						log.Printf("⚠️ [AUTH] Failed to upsert user %s: %v", uid, dbErr)
					}
				}(userID, email, name)
			}
		} else if fetchErr != nil {
			fmt.Printf("⚠️ AUTH: Could not fetch Clerk user profile for %s: %v\n", userID, fetchErr)
		}

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
			Leeway: 5 * time.Second,
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
