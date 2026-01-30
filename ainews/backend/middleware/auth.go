package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"ainews/database"
	"ainews/models"

	"github.com/gin-gonic/gin"
)

const (
	JournalistContextKey = "journalist"
	APIKeyHeader         = "X-API-Key"
	AdminAPIKeyEnvVar    = "ADMIN_API_KEY"
)

// HashAPIKey creates a SHA-256 hash of the API key
func HashAPIKey(apiKey string) string {
	hash := sha256.Sum256([]byte(apiKey))
	return hex.EncodeToString(hash[:])
}

// APIKeyAuth middleware validates API keys for journalist endpoints
func APIKeyAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader(APIKeyHeader)
		if apiKey == "" {
			// Also check Authorization header (Bearer token format)
			authHeader := c.GetHeader("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				apiKey = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}

		if apiKey == "" {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Error: "API key required",
				Code:  "MISSING_API_KEY",
			})
			c.Abort()
			return
		}

		// Hash the provided API key
		keyHash := HashAPIKey(apiKey)

		// Look up journalist by API key hash
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var journalist models.Journalist
		var twitterHandle *string
		err := database.DB.QueryRow(ctx, `
			SELECT id, name, created_at, active, post_count, verified, twitter_handle 
			FROM journalists 
			WHERE api_key_hash = $1
		`, keyHash).Scan(
			&journalist.ID,
			&journalist.Name,
			&journalist.CreatedAt,
			&journalist.Active,
			&journalist.PostCount,
			&journalist.Verified,
			&twitterHandle,
		)

		if twitterHandle != nil {
			journalist.TwitterHandle = *twitterHandle
		}

		if err != nil {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Error: "Invalid API key",
				Code:  "INVALID_API_KEY",
			})
			c.Abort()
			return
		}

		if !journalist.Active {
			c.JSON(http.StatusForbidden, models.ErrorResponse{
				Error: "Journalist account is deactivated",
				Code:  "ACCOUNT_DEACTIVATED",
			})
			c.Abort()
			return
		}

		if !journalist.Verified {
			c.JSON(http.StatusForbidden, models.ErrorResponse{
				Error: "Journalist account not verified. Please post your verification code on Twitter and call POST /api/journalists/verify",
				Code:  "NOT_VERIFIED",
			})
			c.Abort()
			return
		}

		// Store journalist in context
		c.Set(JournalistContextKey, &journalist)
		c.Next()
	}
}

// AdminAuth middleware validates admin API key
func AdminAuth(adminAPIKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader(APIKeyHeader)
		if apiKey == "" {
			authHeader := c.GetHeader("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				apiKey = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}

		if apiKey == "" || apiKey != adminAPIKey {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Error: "Admin API key required",
				Code:  "INVALID_ADMIN_KEY",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// GetJournalist retrieves the journalist from context
func GetJournalist(c *gin.Context) *models.Journalist {
	journalist, exists := c.Get(JournalistContextKey)
	if !exists {
		return nil
	}
	return journalist.(*models.Journalist)
}
