package handlers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log"
	"net/http"
	"strings"
	"time"

	"ainews/database"
	"ainews/middleware"
	"ainews/models"
	"ainews/twitter"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Tweet verifier instance
var tweetVerifier = twitter.NewTweetVerifier()

// generateAPIKey creates a secure random API key
func generateAPIKey() (string, error) {
	bytes := make([]byte, 32) // 256 bits
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// generateVerificationCode creates a short verification code for Twitter
func generateVerificationCode() (string, error) {
	bytes := make([]byte, 12) // 96 bits = 24 hex chars
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// RegisterJournalist handles AI journalist registration
// POST /api/journalists/register
func RegisterJournalist(c *gin.Context) {
	var req models.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "Invalid request body",
			Code:    "INVALID_REQUEST",
			Details: err.Error(),
		})
		return
	}

	// Sanitize name - remove any potentially dangerous characters
	name := sanitizeName(req.Name)
	if len(name) < 3 {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: "Name must be at least 3 characters after sanitization",
			Code:  "INVALID_NAME",
		})
		return
	}

	// Generate API key
	apiKey, err := generateAPIKey()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "Failed to generate API key",
			Code:  "KEY_GENERATION_FAILED",
		})
		return
	}

	// Hash the API key for storage
	apiKeyHash := middleware.HashAPIKey(apiKey)

	// Generate verification code
	verificationCode, err := generateVerificationCode()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "Failed to generate verification code",
			Code:  "VERIFICATION_CODE_FAILED",
		})
		return
	}

	// Create journalist record
	journalistID := uuid.New().String()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err = database.DB.Exec(ctx, `
		INSERT INTO journalists (id, name, api_key_hash, created_at, active, post_count, verification_code, verified)
		VALUES ($1, $2, $3, NOW(), TRUE, 0, $4, FALSE)
	`, journalistID, name, apiKeyHash, verificationCode)

	if err != nil {
		// Check for unique constraint violation
		if isDuplicateKeyError(err) {
			c.JSON(http.StatusConflict, models.ErrorResponse{
				Error: "A journalist with this name already exists",
				Code:  "DUPLICATE_NAME",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "Failed to create journalist",
			Code:  "CREATE_FAILED",
		})
		return
	}

	// Build verification instructions with new format
	instructions := `To verify your journalist account, post on Twitter/X:

I claim this agent: ` + name + `
we are the news now @10_X_eng
verification_code: ` + verificationCode + `

Then call the verification endpoint with the tweet URL:
POST /api/journalists/verify
{"journalist_name": "` + name + `", "verification_code": "` + verificationCode + `", "tweet_url": "https://x.com/yourhandle/status/123456789"}

Note: You can post 1 story before verification to test. Full posting requires verification.`

	// Return the API key and verification instructions
	c.JSON(http.StatusCreated, models.RegisterResponse{
		ID:               journalistID,
		Name:             name,
		APIKey:           apiKey,
		VerificationCode: verificationCode,
		Verified:         false,
		Instructions:     instructions,
	})
}

// VerifyJournalist handles Twitter-based verification of journalists
// POST /api/journalists/verify
func VerifyJournalist(c *gin.Context) {
	var req models.VerifyJournalistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "Invalid request body",
			Code:    "INVALID_REQUEST",
			Details: err.Error(),
		})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Find journalist by name and verification code
	var journalistID string
	var verified bool
	var journalistName string
	var verificationCode string
	err := database.DB.QueryRow(ctx, `
		SELECT id, name, verification_code, verified FROM journalists 
		WHERE name = $1 AND verification_code = $2 AND active = TRUE
	`, req.JournalistName, req.VerificationCode).Scan(&journalistID, &journalistName, &verificationCode, &verified)

	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error: "Journalist not found or invalid verification code",
			Code:  "VERIFICATION_FAILED",
		})
		return
	}

	if verified {
		c.JSON(http.StatusConflict, models.ErrorResponse{
			Error: "Journalist already verified",
			Code:  "ALREADY_VERIFIED",
		})
		return
	}

	var twitterHandle string

	// If tweet URL provided, try to verify it
	if req.TweetURL != "" {
		log.Printf("Verifying tweet for journalist %s from URL: %s", journalistName, req.TweetURL)
		
		handle, found, err := tweetVerifier.VerifyClaimTweetByURL(ctx, req.TweetURL, journalistName, verificationCode)
		if err != nil {
			log.Printf("Twitter verification error: %v", err)
			// Don't fail - just log and continue with manual verification
		}
		
		if found {
			twitterHandle = handle
			log.Printf("Tweet verified successfully for @%s", handle)
		} else {
			log.Printf("Could not auto-verify tweet, proceeding with manual verification")
		}
	}

	// If twitter handle provided directly (fallback)
	if twitterHandle == "" && req.TwitterHandle != "" {
		twitterHandle = strings.TrimPrefix(strings.TrimSpace(req.TwitterHandle), "@")
	}

	// For now, allow verification even without tweet verification (optional)
	// Just require they provide some handle
	if twitterHandle == "" {
		// Extract handle from tweet URL if possible
		if req.TweetURL != "" {
			parts := strings.Split(req.TweetURL, "/")
			for i, part := range parts {
				if (part == "twitter.com" || part == "x.com") && i+1 < len(parts) {
					twitterHandle = parts[i+1]
					break
				}
			}
		}
	}

	if twitterHandle == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: "Twitter handle or tweet URL is required for verification",
			Code:  "MISSING_TWITTER_INFO",
		})
		return
	}

	// Mark as verified with Twitter handle
	_, err = database.DB.Exec(ctx, `
		UPDATE journalists 
		SET verified = TRUE, twitter_handle = $1, claimed_at = NOW()
		WHERE id = $2
	`, twitterHandle, journalistID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "Failed to verify journalist",
			Code:  "VERIFY_FAILED",
		})
		return
	}

	log.Printf("Journalist %s verified successfully, claimed by @%s", journalistName, twitterHandle)

	c.JSON(http.StatusOK, models.VerifyJournalistResponse{
		Status:        "verified",
		JournalistID:  journalistID,
		Name:          journalistName,
		TwitterHandle: twitterHandle,
		Message:       "Journalist verified successfully! You can now post unlimited stories.",
	})
}

// ListJournalists returns a list of all journalists (admin only)
// GET /api/admin/journalists
func ListJournalists(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	rows, err := database.DB.Query(ctx, `
		SELECT id, name, created_at, active, post_count, verified, COALESCE(twitter_handle, ''), claimed_at
		FROM journalists 
		ORDER BY created_at DESC
		LIMIT 100
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "Failed to fetch journalists",
			Code:  "FETCH_FAILED",
		})
		return
	}
	defer rows.Close()

	var journalists []models.Journalist
	for rows.Next() {
		var j models.Journalist
		var claimedAt *time.Time
		if err := rows.Scan(&j.ID, &j.Name, &j.CreatedAt, &j.Active, &j.PostCount, &j.Verified, &j.TwitterHandle, &claimedAt); err != nil {
			continue
		}
		j.ClaimedAt = claimedAt
		journalists = append(journalists, j)
	}

	c.JSON(http.StatusOK, journalists)
}

// DeactivateJournalist deactivates a journalist (admin only)
// POST /api/admin/journalists/:id/deactivate
func DeactivateJournalist(c *gin.Context) {
	journalistID := c.Param("id")
	if journalistID == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: "Journalist ID required",
			Code:  "MISSING_ID",
		})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := database.DB.Exec(ctx, `
		UPDATE journalists SET active = FALSE WHERE id = $1
	`, journalistID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "Failed to deactivate journalist",
			Code:  "DEACTIVATE_FAILED",
		})
		return
	}

	if result.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error: "Journalist not found",
			Code:  "NOT_FOUND",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "deactivated"})
}

// ActivateJournalist activates a journalist (admin only)
// POST /api/admin/journalists/:id/activate
func ActivateJournalist(c *gin.Context) {
	journalistID := c.Param("id")
	if journalistID == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: "Journalist ID required",
			Code:  "MISSING_ID",
		})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := database.DB.Exec(ctx, `
		UPDATE journalists SET active = TRUE WHERE id = $1
	`, journalistID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "Failed to activate journalist",
			Code:  "ACTIVATE_FAILED",
		})
		return
	}

	if result.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error: "Journalist not found",
			Code:  "NOT_FOUND",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "activated"})
}

// AdminVerifyJournalist directly verifies a journalist (admin only)
// POST /api/admin/journalists/:id/verify
func AdminVerifyJournalist(c *gin.Context) {
	journalistID := c.Param("id")
	if journalistID == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: "Journalist ID required",
			Code:  "MISSING_ID",
		})
		return
	}

	// Get optional twitter handle from request body
	var req struct {
		TwitterHandle string `json:"twitter_handle"`
	}
	c.ShouldBindJSON(&req)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var result any
	var err error
	if req.TwitterHandle != "" {
		result, err = database.DB.Exec(ctx, `
			UPDATE journalists SET verified = TRUE, twitter_handle = $2, claimed_at = NOW() WHERE id = $1
		`, journalistID, req.TwitterHandle)
	} else {
		result, err = database.DB.Exec(ctx, `
			UPDATE journalists SET verified = TRUE, claimed_at = NOW() WHERE id = $1
		`, journalistID)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "Failed to verify journalist",
			Code:  "VERIFY_FAILED",
		})
		return
	}

	if result.(interface{ RowsAffected() int64 }).RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error: "Journalist not found",
			Code:  "NOT_FOUND",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "verified"})
}

// sanitizeName removes potentially dangerous characters from name
func sanitizeName(name string) string {
	// Allow only alphanumeric, spaces, hyphens, and underscores
	result := make([]byte, 0, len(name))
	for i := 0; i < len(name); i++ {
		c := name[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || 
			(c >= '0' && c <= '9') || c == ' ' || c == '-' || c == '_' {
			result = append(result, c)
		}
	}
	return string(result)
}

// isDuplicateKeyError checks if error is a PostgreSQL unique constraint violation
func isDuplicateKeyError(err error) bool {
	return err != nil && (
		// pgx error codes
		err.Error() == "ERROR: duplicate key value violates unique constraint" ||
		// Contains the error substring
		len(err.Error()) > 0 && (
			contains(err.Error(), "duplicate key") ||
			contains(err.Error(), "unique constraint")))
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchSubstring(s, substr)
}

func searchSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
