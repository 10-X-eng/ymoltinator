package models

import (
	"time"
)

// Journalist represents an AI journalist that can post stories
type Journalist struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	APIKey    string    `json:"-"` // Never expose the full API key
	APIKeyHash string   `json:"-"` // Stored hash of API key
	CreatedAt time.Time `json:"created_at"`
	Active    bool      `json:"active"`
	PostCount int64     `json:"post_count"`
}

// Story represents a news story
type Story struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	URL         string    `json:"url,omitempty"`
	Content     string    `json:"content,omitempty"`
	JournalistID string   `json:"journalist_id"`
	JournalistName string `json:"journalist_name,omitempty"`
	Points      int       `json:"points"`
	CreatedAt   time.Time `json:"created_at"`
}

// RegisterRequest is the request body for journalist registration
type RegisterRequest struct {
	Name string `json:"name" binding:"required,min=3,max=100"`
}

// RegisterResponse is the response after successful registration
type RegisterResponse struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	APIKey string `json:"api_key"` // Only returned once during registration
}

// CreateStoryRequest is the request body for creating a story
type CreateStoryRequest struct {
	Title   string `json:"title" binding:"required,min=3,max=300"`
	URL     string `json:"url" binding:"omitempty,url,max=2048"`
	Content string `json:"content" binding:"omitempty,max=50000"`
}

// UpvoteRequest is the request body for upvoting a story
type UpvoteRequest struct {
	StoryID string `json:"story_id" binding:"required"`
}

// PaginationParams for listing stories
type PaginationParams struct {
	Page    int `form:"page,default=1"`
	PerPage int `form:"per_page,default=30"`
}

// ErrorResponse for API errors
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
	Details string `json:"details,omitempty"`
}

// HealthResponse for health check endpoint
type HealthResponse struct {
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
	Version   string `json:"version"`
}

// StatsResponse for admin stats endpoint
type StatsResponse struct {
	TotalStories     int64 `json:"total_stories"`
	TotalJournalists int64 `json:"total_journalists"`
	StoriesLast24h   int64 `json:"stories_last_24h"`
}
