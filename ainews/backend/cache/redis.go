package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// Cache keys
const (
	KeyStoriesList   = "stories:list:%d:%d" // page:perpage
	KeyStory         = "story:%s"           // story_id
	KeyStats         = "stats:global"
	
	// TTLs
	TTLStoriesList   = 30 * time.Second     // Short TTL for list freshness
	TTLStory         = 2 * time.Minute      // Individual stories cached longer
	TTLStats         = 1 * time.Minute
)

// Client wraps Redis client
type Client struct {
	rdb *redis.Client
}

var instance *Client

// InitRedis initializes Redis connection
func InitRedis(redisURL string) error {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return fmt.Errorf("failed to parse redis URL: %w", err)
	}
	
	// Connection pool settings for high performance
	opt.PoolSize = 100
	opt.MinIdleConns = 10
	opt.MaxRetries = 3
	opt.DialTimeout = 5 * time.Second
	opt.ReadTimeout = 3 * time.Second
	opt.WriteTimeout = 3 * time.Second
	
	rdb := redis.NewClient(opt)
	
	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := rdb.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("failed to connect to redis: %w", err)
	}
	
	instance = &Client{rdb: rdb}
	log.Println("Redis connection established")
	return nil
}

// Close closes Redis connection
func Close() {
	if instance != nil && instance.rdb != nil {
		instance.rdb.Close()
	}
}

// Get retrieves and deserializes a cached value
func Get[T any](ctx context.Context, key string) (*T, error) {
	if instance == nil {
		return nil, nil
	}
	
	data, err := instance.rdb.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // Cache miss
		}
		return nil, err
	}
	
	var result T
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	
	return &result, nil
}

// Set serializes and caches a value with TTL
func Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	if instance == nil {
		return nil
	}
	
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	
	return instance.rdb.Set(ctx, key, data, ttl).Err()
}

// Delete removes a key from cache
func Delete(ctx context.Context, key string) error {
	if instance == nil {
		return nil
	}
	return instance.rdb.Del(ctx, key).Err()
}

// DeletePattern removes keys matching a pattern
func DeletePattern(ctx context.Context, pattern string) error {
	if instance == nil {
		return nil
	}
	
	iter := instance.rdb.Scan(ctx, 0, pattern, 100).Iterator()
	var keys []string
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}
	
	if err := iter.Err(); err != nil {
		return err
	}
	
	if len(keys) > 0 {
		return instance.rdb.Del(ctx, keys...).Err()
	}
	
	return nil
}

// InvalidateStories invalidates all story-related caches
func InvalidateStories(ctx context.Context) error {
	if instance == nil {
		return nil
	}
	
	// Delete all list caches
	if err := DeletePattern(ctx, "stories:list:*"); err != nil {
		log.Printf("Warning: failed to invalidate stories list cache: %v", err)
	}
	
	// Delete stats cache
	if err := Delete(ctx, KeyStats); err != nil {
		log.Printf("Warning: failed to invalidate stats cache: %v", err)
	}
	
	return nil
}

// InvalidateStory invalidates a specific story cache
func InvalidateStory(ctx context.Context, storyID string) error {
	if instance == nil {
		return nil
	}
	
	key := fmt.Sprintf(KeyStory, storyID)
	if err := Delete(ctx, key); err != nil {
		log.Printf("Warning: failed to invalidate story cache %s: %v", storyID, err)
	}
	
	// Also invalidate list caches since story data changed
	return InvalidateStories(ctx)
}

// GetStoriesListKey returns the cache key for a paginated stories list
func GetStoriesListKey(page, perPage int) string {
	return fmt.Sprintf(KeyStoriesList, page, perPage)
}

// GetStoryKey returns the cache key for a single story
func GetStoryKey(storyID string) string {
	return fmt.Sprintf(KeyStory, storyID)
}

// Healthy checks if Redis is responsive
func Healthy(ctx context.Context) bool {
	if instance == nil {
		return false
	}
	return instance.rdb.Ping(ctx).Err() == nil
}
