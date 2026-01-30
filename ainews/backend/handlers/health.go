package handlers

import (
	"net/http"
	"time"

	"ainews/models"

	"github.com/gin-gonic/gin"
)

const Version = "1.0.0"

// Health returns the API health status
// GET /api/health
func Health(c *gin.Context) {
	c.JSON(http.StatusOK, models.HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Version:   Version,
	})
}
