package handlers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"ainews/cache"
	"ainews/database"
	"ainews/middleware"
	"ainews/models"
	"ainews/moderation"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

var contentFilter *moderation.Filter

func init() {
	contentFilter = moderation.NewFilter()
}

// CreateStory handles story creation by authenticated journalists
// POST /api/stories
func CreateStory(c *gin.Context) {
	journalist := middleware.GetJournalist(c)
	if journalist == nil {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error: "Authentication required",
			Code:  "AUTH_REQUIRED",
		})
		return
	}

	var req models.CreateStoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "Invalid request body",
			Code:    "INVALID_REQUEST",
			Details: err.Error(),
		})
		return
	}

	// Validate content doesn't have prohibited words
	if valid, reason := contentFilter.ValidateContent(req.Title, req.Content, req.URL); !valid {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "Content rejected by moderation",
			Code:    "CONTENT_REJECTED",
			Details: reason,
		})
		return
	}

	// Must have either URL or content
	if req.URL == "" && req.Content == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: "Either URL or content is required",
			Code:  "MISSING_CONTENT",
		})
		return
	}

	// Create story
	storyID := uuid.New().String()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Start transaction
	tx, err := database.DB.Begin(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "Failed to start transaction",
			Code:  "TX_START_FAILED",
		})
		return
	}
	defer tx.Rollback(ctx)

	// Insert story
	_, err = tx.Exec(ctx, `
		INSERT INTO stories (id, title, url, content, journalist_id, points, created_at)
		VALUES ($1, $2, $3, $4, $5, 1, NOW())
	`, storyID, req.Title, nullString(req.URL), nullString(req.Content), journalist.ID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "Failed to create story",
			Code:  "CREATE_FAILED",
		})
		return
	}

	// Increment journalist post count
	_, err = tx.Exec(ctx, `
		UPDATE journalists SET post_count = post_count + 1 WHERE id = $1
	`, journalist.ID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "Failed to update post count",
			Code:  "UPDATE_FAILED",
		})
		return
	}

	// Record rate limit action
	_, err = tx.Exec(ctx, `
		INSERT INTO rate_limits (id, journalist_id, action, created_at)
		VALUES ($1, $2, 'create_story', NOW())
	`, uuid.New().String(), journalist.ID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "Failed to record action",
			Code:  "RECORD_FAILED",
		})
		return
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "Failed to commit transaction",
			Code:  "TX_COMMIT_FAILED",
		})
		return
	}

	// Invalidate cache after creating story
	cache.InvalidateStories(ctx)

	c.JSON(http.StatusCreated, models.Story{
		ID:           storyID,
		Title:        req.Title,
		URL:          req.URL,
		Content:      req.Content,
		JournalistID: journalist.ID,
		JournalistName: journalist.Name,
		Points:       1,
		CreatedAt:    time.Now(),
	})
}

// ListStories returns paginated list of stories
// GET /api/stories
func ListStories(c *gin.Context) {
	var params models.PaginationParams
	if err := c.ShouldBindQuery(&params); err != nil {
		params.Page = 1
		params.PerPage = 30
	}

	// Validate pagination
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PerPage < 1 || params.PerPage > 100 {
		params.PerPage = 30
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Try cache first
	cacheKey := cache.GetStoriesListKey(params.Page, params.PerPage)
	if cached, err := cache.Get[[]models.Story](ctx, cacheKey); err == nil && cached != nil {
		c.Header("X-Cache", "HIT")
		c.JSON(http.StatusOK, *cached)
		return
	}

	// Cache miss - fetch from database
	offset := (params.Page - 1) * params.PerPage

	rows, err := database.DB.Query(ctx, `
		SELECT s.id, s.title, COALESCE(s.url, ''), COALESCE(s.content, ''), 
			   s.journalist_id, j.name, s.points, s.created_at
		FROM stories s
		JOIN journalists j ON s.journalist_id = j.id
		ORDER BY s.created_at DESC
		LIMIT $1 OFFSET $2
	`, params.PerPage, offset)

	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "Failed to fetch stories",
			Code:  "FETCH_FAILED",
		})
		return
	}
	defer rows.Close()

	stories := make([]models.Story, 0)
	for rows.Next() {
		var s models.Story
		if err := rows.Scan(&s.ID, &s.Title, &s.URL, &s.Content, 
			&s.JournalistID, &s.JournalistName, &s.Points, &s.CreatedAt); err != nil {
			continue
		}
		// Truncate content for list view
		if len(s.Content) > 500 {
			s.Content = s.Content[:500] + "..."
		}
		stories = append(stories, s)
	}

	// Cache the result
	if err := cache.Set(ctx, cacheKey, stories, cache.TTLStoriesList); err != nil {
		// Log but don't fail
		fmt.Printf("Warning: failed to cache stories list: %v\n", err)
	}

	c.Header("X-Cache", "MISS")
	c.JSON(http.StatusOK, stories)
}

// GetStory returns a single story by ID
// GET /api/stories/:id
func GetStory(c *gin.Context) {
	storyID := c.Param("id")
	if storyID == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: "Story ID required",
			Code:  "MISSING_ID",
		})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Try cache first
	cacheKey := cache.GetStoryKey(storyID)
	if cached, err := cache.Get[models.Story](ctx, cacheKey); err == nil && cached != nil {
		c.Header("X-Cache", "HIT")
		c.JSON(http.StatusOK, *cached)
		return
	}

	// Cache miss - fetch from database
	var s models.Story
	err := database.DB.QueryRow(ctx, `
		SELECT s.id, s.title, COALESCE(s.url, ''), COALESCE(s.content, ''),
			   s.journalist_id, j.name, s.points, s.created_at
		FROM stories s
		JOIN journalists j ON s.journalist_id = j.id
		WHERE s.id = $1
	`, storyID).Scan(&s.ID, &s.Title, &s.URL, &s.Content, 
		&s.JournalistID, &s.JournalistName, &s.Points, &s.CreatedAt)

	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error: "Story not found",
			Code:  "NOT_FOUND",
		})
		return
	}

	// Cache the result
	if err := cache.Set(ctx, cacheKey, s, cache.TTLStory); err != nil {
		fmt.Printf("Warning: failed to cache story: %v\n", err)
	}

	c.Header("X-Cache", "MISS")
	c.JSON(http.StatusOK, s)
}

// UpvoteStory handles story upvoting
// POST /api/stories/:id/upvote
func UpvoteStory(c *gin.Context) {
	storyID := c.Param("id")
	if storyID == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: "Story ID required",
			Code:  "MISSING_ID",
		})
		return
	}

	// Get client IP for rate limiting upvotes
	ip := getClientIP(c)
	ipHash := hashIP(ip)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Check if already upvoted
	var exists bool
	err := database.DB.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM upvotes WHERE story_id = $1 AND ip_hash = $2)
	`, storyID, ipHash).Scan(&exists)

	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "Failed to check upvote status",
			Code:  "CHECK_FAILED",
		})
		return
	}

	if exists {
		c.JSON(http.StatusConflict, models.ErrorResponse{
			Error: "Already upvoted",
			Code:  "ALREADY_UPVOTED",
		})
		return
	}

	// Start transaction
	tx, err := database.DB.Begin(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "Failed to start transaction",
			Code:  "TX_START_FAILED",
		})
		return
	}
	defer tx.Rollback(ctx)

	// Record upvote
	_, err = tx.Exec(ctx, `
		INSERT INTO upvotes (id, story_id, ip_hash, created_at)
		VALUES ($1, $2, $3, NOW())
	`, uuid.New().String(), storyID, ipHash)

	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "Failed to record upvote",
			Code:  "UPVOTE_FAILED",
		})
		return
	}

	// Increment points
	result, err := tx.Exec(ctx, `
		UPDATE stories SET points = points + 1 WHERE id = $1
	`, storyID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "Failed to update points",
			Code:  "UPDATE_FAILED",
		})
		return
	}

	if result.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error: "Story not found",
			Code:  "NOT_FOUND",
		})
		return
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "Failed to commit transaction",
			Code:  "TX_COMMIT_FAILED",
		})
		return
	}

	// Invalidate story cache after upvote
	cache.InvalidateStory(ctx, storyID)

	c.JSON(http.StatusOK, gin.H{"status": "upvoted"})
}

// DeleteStory deletes a story (admin only)
// DELETE /api/admin/stories/:id
func DeleteStory(c *gin.Context) {
	storyID := c.Param("id")
	if storyID == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: "Story ID required",
			Code:  "MISSING_ID",
		})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := database.DB.Exec(ctx, `DELETE FROM stories WHERE id = $1`, storyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "Failed to delete story",
			Code:  "DELETE_FAILED",
		})
		return
	}

	if result.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error: "Story not found",
			Code:  "NOT_FOUND",
		})
		return
	}

	// Invalidate caches after delete
	cache.InvalidateStory(ctx, storyID)

	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

// GetStats returns site statistics (admin only)
// GET /api/admin/stats
func GetStats(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Try cache first
	if cached, err := cache.Get[models.StatsResponse](ctx, cache.KeyStats); err == nil && cached != nil {
		c.Header("X-Cache", "HIT")
		c.JSON(http.StatusOK, *cached)
		return
	}

	var stats models.StatsResponse

	// Get total stories
	database.DB.QueryRow(ctx, `SELECT COUNT(*) FROM stories`).Scan(&stats.TotalStories)

	// Get total journalists
	database.DB.QueryRow(ctx, `SELECT COUNT(*) FROM journalists`).Scan(&stats.TotalJournalists)

	// Get stories in last 24h
	database.DB.QueryRow(ctx, `
		SELECT COUNT(*) FROM stories WHERE created_at > NOW() - INTERVAL '24 hours'
	`).Scan(&stats.StoriesLast24h)

	// Cache the result
	if err := cache.Set(ctx, cache.KeyStats, stats, cache.TTLStats); err != nil {
		fmt.Printf("Warning: failed to cache stats: %v\n", err)
	}

	c.Header("X-Cache", "MISS")
	c.JSON(http.StatusOK, stats)
}

// Helper functions
func nullString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func getClientIP(c *gin.Context) string {
	xff := c.GetHeader("X-Forwarded-For")
	if xff != "" {
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}
	xri := c.GetHeader("X-Real-IP")
	if xri != "" {
		return xri
	}
	ip, _, err := net.SplitHostPort(c.Request.RemoteAddr)
	if err != nil {
		return c.Request.RemoteAddr
	}
	return ip
}

func hashIP(ip string) string {
	hash := sha256.Sum256([]byte(ip))
	return hex.EncodeToString(hash[:])
}
