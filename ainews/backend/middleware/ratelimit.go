package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"ainews/database"
	"ainews/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/time/rate"
)

// RateLimiter manages rate limiting for different types of requests
type RateLimiter struct {
	readers      map[string]*rate.Limiter
	readersMu    sync.RWMutex
	readerLimit  rate.Limit
	readerBurst  int
	writerPostPerMin int
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(readerRPM int, writerPostPerMin int) *RateLimiter {
	return &RateLimiter{
		readers:          make(map[string]*rate.Limiter),
		readerLimit:      rate.Limit(float64(readerRPM) / 60.0), // Convert RPM to per-second
		readerBurst:      readerRPM / 10, // Allow short bursts
		writerPostPerMin: writerPostPerMin,
	}
}

// getClientIP extracts the real client IP from the request
func getClientIP(c *gin.Context) string {
	// Check X-Forwarded-For header (from nginx)
	xff := c.GetHeader("X-Forwarded-For")
	if xff != "" {
		// Take the first IP in the chain
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}

	// Check X-Real-IP header
	xri := c.GetHeader("X-Real-IP")
	if xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	ip, _, err := net.SplitHostPort(c.Request.RemoteAddr)
	if err != nil {
		return c.Request.RemoteAddr
	}
	return ip
}

// hashIP creates a hash of the IP for storage
func hashIP(ip string) string {
	hash := sha256.Sum256([]byte(ip))
	return hex.EncodeToString(hash[:])
}

// getReaderLimiter returns or creates a rate limiter for a reader IP
func (rl *RateLimiter) getReaderLimiter(ip string) *rate.Limiter {
	rl.readersMu.RLock()
	limiter, exists := rl.readers[ip]
	rl.readersMu.RUnlock()

	if exists {
		return limiter
	}

	rl.readersMu.Lock()
	defer rl.readersMu.Unlock()

	// Double-check after acquiring write lock
	if limiter, exists = rl.readers[ip]; exists {
		return limiter
	}

	limiter = rate.NewLimiter(rl.readerLimit, rl.readerBurst)
	rl.readers[ip] = limiter
	return limiter
}

// ReaderRateLimit middleware limits read requests by IP
func (rl *RateLimiter) ReaderRateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := getClientIP(c)
		limiter := rl.getReaderLimiter(ip)

		if !limiter.Allow() {
			c.JSON(http.StatusTooManyRequests, models.ErrorResponse{
				Error: "Rate limit exceeded. Please slow down.",
				Code:  "RATE_LIMIT_EXCEEDED",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// WriterRateLimit middleware limits post creation by journalist
func (rl *RateLimiter) WriterRateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		journalist := GetJournalist(c)
		if journalist == nil {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Error: "Authentication required",
				Code:  "AUTH_REQUIRED",
			})
			c.Abort()
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Check how many posts in the last minute
		var count int
		err := database.DB.QueryRow(ctx, `
			SELECT COUNT(*) FROM rate_limits 
			WHERE journalist_id = $1 
			AND action = 'create_story' 
			AND created_at > NOW() - INTERVAL '1 minute'
		`, journalist.ID).Scan(&count)

		if err != nil {
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{
				Error: "Failed to check rate limit",
				Code:  "RATE_CHECK_FAILED",
			})
			c.Abort()
			return
		}

		if count >= rl.writerPostPerMin {
			c.JSON(http.StatusTooManyRequests, models.ErrorResponse{
				Error: "Post rate limit exceeded. You can only post " + 
					string(rune('0'+rl.writerPostPerMin)) + " story per minute.",
				Code:  "POST_RATE_LIMIT_EXCEEDED",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RecordWriteAction records a write action for rate limiting
func RecordWriteAction(ctx context.Context, journalistID, action string) error {
	_, err := database.DB.Exec(ctx, `
		INSERT INTO rate_limits (id, journalist_id, action, created_at)
		VALUES ($1, $2, $3, NOW())
	`, uuid.New().String(), journalistID, action)
	return err
}

// CleanupRateLimits periodically cleans up old rate limit entries
func (rl *RateLimiter) CleanupRateLimits(interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			// Clean up in-memory limiters (remove inactive ones)
			rl.readersMu.Lock()
			// Reset the map periodically to prevent memory growth
			// Active clients will recreate their limiters
			if len(rl.readers) > 10000 {
				rl.readers = make(map[string]*rate.Limiter)
			}
			rl.readersMu.Unlock()

			// Clean up database rate limits
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			database.DB.Exec(ctx, "SELECT cleanup_old_rate_limits()")
			cancel()
		}
	}()
}
