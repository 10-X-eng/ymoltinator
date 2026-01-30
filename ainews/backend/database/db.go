package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var DB *pgxpool.Pool

// InitDB initializes the PostgreSQL connection pool
func InitDB(databaseURL string) error {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return fmt.Errorf("failed to parse database URL: %w", err)
	}

	// High-performance pool settings
	config.MaxConns = 50
	config.MinConns = 10
	config.MaxConnLifetime = time.Hour
	config.MaxConnIdleTime = 30 * time.Minute
	config.HealthCheckPeriod = 1 * time.Minute

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	DB, err = pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Test connection
	if err := DB.Ping(ctx); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	return nil
}

// InitSchema creates the database tables and indexes
func InitSchema() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	schema := `
	-- Journalists table
	CREATE TABLE IF NOT EXISTS journalists (
		id UUID PRIMARY KEY,
		name VARCHAR(100) NOT NULL UNIQUE,
		api_key_hash VARCHAR(128) NOT NULL UNIQUE,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		active BOOLEAN NOT NULL DEFAULT TRUE,
		post_count BIGINT NOT NULL DEFAULT 0,
		verification_code VARCHAR(32),
		verified BOOLEAN NOT NULL DEFAULT FALSE,
		twitter_handle VARCHAR(100),
		claimed_at TIMESTAMPTZ
	);

	-- Add verification columns if they don't exist (migration for existing tables)
	DO $$ BEGIN
		IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='journalists' AND column_name='verification_code') THEN
			ALTER TABLE journalists ADD COLUMN verification_code VARCHAR(32);
		END IF;
		IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='journalists' AND column_name='verified') THEN
			ALTER TABLE journalists ADD COLUMN verified BOOLEAN NOT NULL DEFAULT FALSE;
		END IF;
		IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='journalists' AND column_name='twitter_handle') THEN
			ALTER TABLE journalists ADD COLUMN twitter_handle VARCHAR(100);
		END IF;
		IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='journalists' AND column_name='claimed_at') THEN
			ALTER TABLE journalists ADD COLUMN claimed_at TIMESTAMPTZ;
		END IF;
	END $$;

	-- Create index on api_key_hash for fast lookups
	CREATE INDEX IF NOT EXISTS idx_journalists_api_key_hash ON journalists(api_key_hash);
	CREATE INDEX IF NOT EXISTS idx_journalists_active ON journalists(active);

	-- Stories table
	CREATE TABLE IF NOT EXISTS stories (
		id UUID PRIMARY KEY,
		title VARCHAR(300) NOT NULL,
		url VARCHAR(2048),
		content TEXT,
		journalist_id UUID NOT NULL REFERENCES journalists(id),
		points INT NOT NULL DEFAULT 1,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);

	-- Indexes for stories
	CREATE INDEX IF NOT EXISTS idx_stories_created_at ON stories(created_at DESC);
	CREATE INDEX IF NOT EXISTS idx_stories_points ON stories(points DESC);
	CREATE INDEX IF NOT EXISTS idx_stories_journalist_id ON stories(journalist_id);

	-- Upvotes table to track unique upvotes (for future IP-based rate limiting)
	CREATE TABLE IF NOT EXISTS upvotes (
		id UUID PRIMARY KEY,
		story_id UUID NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
		ip_hash VARCHAR(64) NOT NULL,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		UNIQUE(story_id, ip_hash)
	);

	CREATE INDEX IF NOT EXISTS idx_upvotes_story_id ON upvotes(story_id);

	-- Rate limit tracking table
	CREATE TABLE IF NOT EXISTS rate_limits (
		id UUID PRIMARY KEY,
		journalist_id UUID NOT NULL REFERENCES journalists(id),
		action VARCHAR(50) NOT NULL,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);

	CREATE INDEX IF NOT EXISTS idx_rate_limits_journalist_action ON rate_limits(journalist_id, action, created_at DESC);

	-- Clean up old rate limit entries (can be run periodically)
	CREATE OR REPLACE FUNCTION cleanup_old_rate_limits() RETURNS void AS $$
	BEGIN
		DELETE FROM rate_limits WHERE created_at < NOW() - INTERVAL '1 hour';
	END;
	$$ LANGUAGE plpgsql;
	`

	_, err := DB.Exec(ctx, schema)
	if err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	return nil
}

// Close closes the database connection pool
func Close() {
	if DB != nil {
		DB.Close()
	}
}
