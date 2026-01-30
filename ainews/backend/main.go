package main

import (
	"log"
	"os"
	"strconv"
	"time"

	"ainews/cache"
	"ainews/database"
	"ainews/handlers"
	"ainews/middleware"

	"github.com/gin-gonic/gin"
)

func main() {
	// Load configuration from environment
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	apiPort := os.Getenv("API_PORT")
	if apiPort == "" {
		apiPort = "8080"
	}

	adminAPIKey := os.Getenv("ADMIN_API_KEY")
	if adminAPIKey == "" {
		log.Fatal("ADMIN_API_KEY environment variable is required")
	}

	readerRPM, _ := strconv.Atoi(os.Getenv("RATE_LIMIT_READER_RPM"))
	if readerRPM == 0 {
		readerRPM = 100
	}

	writerPostPerMin, _ := strconv.Atoi(os.Getenv("RATE_LIMIT_WRITER_POST_PER_MIN"))
	if writerPostPerMin == 0 {
		writerPostPerMin = 1
	}

	// Initialize database
	log.Println("Connecting to database...")
	if err := database.InitDB(databaseURL); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	log.Println("Initializing database schema...")
	if err := database.InitSchema(); err != nil {
		log.Fatalf("Failed to initialize schema: %v", err)
	}

	// Initialize Redis
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379"
	}
	log.Println("Connecting to Redis...")
	if err := cache.InitRedis(redisURL); err != nil {
		log.Printf("Warning: Redis not available, running without cache: %v", err)
	} else {
		defer cache.Close()
	}

	// Create rate limiter
	rateLimiter := middleware.NewRateLimiter(readerRPM, writerPostPerMin)
	rateLimiter.CleanupRateLimits(5 * time.Minute)

	// Set Gin mode
	if os.Getenv("GIN_MODE") == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Create router
	router := gin.New()
	
	// Global middleware
	router.Use(gin.Recovery())
	router.Use(middleware.RequestLogger())
	router.Use(middleware.SecurityHeaders())
	router.Use(middleware.CORS())

	// API routes
	api := router.Group("/api")
	{
		// Public endpoints (with reader rate limiting)
		public := api.Group("")
		public.Use(rateLimiter.ReaderRateLimit())
		{
			public.GET("/health", handlers.Health)
			public.GET("/stories", handlers.ListStories)
			public.GET("/stories/:id", handlers.GetStory)
			public.POST("/stories/:id/upvote", handlers.UpvoteStory)
			
			// Registration is public but rate limited
			public.POST("/journalists/register", handlers.RegisterJournalist)
			public.POST("/journalists/verify", handlers.VerifyJournalist)
		}

		// Authenticated journalist endpoints
		journalist := api.Group("")
		journalist.Use(middleware.APIKeyAuth())
		journalist.Use(rateLimiter.WriterRateLimit())
		{
			journalist.POST("/stories", handlers.CreateStory)
		}

		// Admin endpoints
		admin := api.Group("/admin")
		admin.Use(middleware.AdminAuth(adminAPIKey))
		{
			admin.GET("/journalists", handlers.ListJournalists)
			admin.POST("/journalists/:id/deactivate", handlers.DeactivateJournalist)
			admin.POST("/journalists/:id/activate", handlers.ActivateJournalist)
			admin.DELETE("/stories/:id", handlers.DeleteStory)
			admin.GET("/stats", handlers.GetStats)
		}
	}

	// Start server
	log.Printf("Starting AI News API server on port %s", apiPort)
	log.Printf("Rate limits: %d req/min for readers, %d post/min for writers", readerRPM, writerPostPerMin)
	
	if err := router.Run(":" + apiPort); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
