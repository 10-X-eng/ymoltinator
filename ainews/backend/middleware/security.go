package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
)

// SecurityHeaders adds security headers to all responses
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Prevent XSS attacks
		c.Header("X-XSS-Protection", "1; mode=block")
		
		// Prevent clickjacking
		c.Header("X-Frame-Options", "DENY")
		
		// Prevent MIME type sniffing
		c.Header("X-Content-Type-Options", "nosniff")
		
		// Referrer policy
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		
		// Content Security Policy
		c.Header("Content-Security-Policy", 
			"default-src 'self'; "+
			"script-src 'self' 'unsafe-inline'; "+
			"style-src 'self' 'unsafe-inline'; "+
			"img-src 'self' data: https:; "+
			"font-src 'self'; "+
			"connect-src 'self'; "+
			"frame-ancestors 'none'")
		
		// Permissions Policy
		c.Header("Permissions-Policy", 
			"accelerometer=(), camera=(), geolocation=(), gyroscope=(), "+
			"magnetometer=(), microphone=(), payment=(), usb=()")
		
		// HSTS (Strict Transport Security)
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		
		c.Next()
	}
}

// CORS handles Cross-Origin Resource Sharing
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		
		// Allow requests from the same domain or no origin (same-site requests)
		if origin != "" {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
		}
		
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", 
			"Accept, Authorization, Content-Type, X-API-Key, X-Requested-With")
		c.Header("Access-Control-Max-Age", "86400")
		
		// Handle preflight requests
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		
		c.Next()
	}
}

// RequestLogger logs requests with timing
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery
		
		c.Next()
		
		latency := time.Since(start)
		status := c.Writer.Status()
		
		// Only log errors and slow requests in production
		if status >= 400 || latency > 500*time.Millisecond {
			if query != "" {
				path = path + "?" + query
			}
			// Log format: status latency method path
			gin.DefaultWriter.Write([]byte(
				time.Now().Format("2006/01/02 15:04:05") + " | " +
				string(rune(status/100+'0')) + string(rune((status/10)%10+'0')) + string(rune(status%10+'0')) + " | " +
				latency.String() + " | " +
				c.Request.Method + " | " +
				path + "\n"))
		}
	}
}
