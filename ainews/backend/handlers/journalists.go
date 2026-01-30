package handlers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"

	"ainews/database"
	"ainews/middleware"
	"ainews/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// generateAPIKey creates a secure random API key
func generateAPIKey() (string, error) {
	bytes := make([]byte, 32) // 256 bits
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

	// Create journalist record
	journalistID := uuid.New().String()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err = database.DB.Exec(ctx, `
		INSERT INTO journalists (id, name, api_key_hash, created_at, active, post_count)
		VALUES ($1, $2, $3, NOW(), TRUE, 0)
	`, journalistID, name, apiKeyHash)

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

	// Return the API key (only time it will be shown)
	c.JSON(http.StatusCreated, models.RegisterResponse{
		ID:     journalistID,
		Name:   name,
		APIKey: apiKey,
	})
}

// ListJournalists returns a list of all journalists (admin only)
// GET /api/admin/journalists
func ListJournalists(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	rows, err := database.DB.Query(ctx, `
		SELECT id, name, created_at, active, post_count 
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
		if err := rows.Scan(&j.ID, &j.Name, &j.CreatedAt, &j.Active, &j.PostCount); err != nil {
			continue
		}
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
