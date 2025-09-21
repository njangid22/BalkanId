package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port                   string
	FrontendURL            string
	JWTSecret              string
	SessionCookieName      string
	SessionTTL             time.Duration
	RateLimitRPS           float64
	DefaultUserQuotaBytes  int64
	MaxUploadBytes         int64
	SupabaseURL            string
	SupabaseAnonKey        string
	SupabaseServiceRoleKey string
	SupabaseDBURL          string
	StorageBucket          string
	RedisURL               string
	OAuthRedirectURL       string
	GoogleClientID         string
	GoogleClientSecret     string
}

func Load() Config {
	return Config{
		Port:                   getEnv("PORT", "8080"),
		FrontendURL:            getEnv("FRONTEND_URL", "http://localhost:3000"),
		JWTSecret:              getEnv("JWT_SECRET", "change-me"),
		SessionCookieName:      getEnv("SESSION_COOKIE_NAME", "vault_session"),
		SessionTTL:             getDuration("SESSION_TTL", 24*time.Hour),
		RateLimitRPS:           getFloat("RATE_LIMIT_RPS", 2),
		DefaultUserQuotaBytes:  getInt("DEFAULT_USER_QUOTA_BYTES", 10485760),
		MaxUploadBytes:         getInt("MAX_UPLOAD_BYTES", 10_485_760),
		SupabaseURL:            os.Getenv("SUPABASE_URL"),
		SupabaseAnonKey:        os.Getenv("SUPABASE_ANON_KEY"),
		SupabaseServiceRoleKey: os.Getenv("SUPABASE_SERVICE_ROLE_KEY"),
		SupabaseDBURL:          os.Getenv("SUPABASE_DB_URL"),
		StorageBucket:          getEnv("STORAGE_BUCKET", "blobs"),
		RedisURL:               getEnv("REDIS_URL", "redis://redis:6379"),
		OAuthRedirectURL:       os.Getenv("OAUTH_REDIRECT_URL"),
		GoogleClientID:         os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret:     os.Getenv("GOOGLE_CLIENT_SECRET"),
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getInt(key string, fallback int64) int64 {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseInt(value, 10, 64); err == nil {
			return parsed
		}
	}
	return fallback
}

func getFloat(key string, fallback float64) float64 {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseFloat(value, 64); err == nil {
			return parsed
		}
	}
	return fallback
}

func getDuration(key string, fallback time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if parsed, err := time.ParseDuration(value); err == nil {
			return parsed
		}
	}
	return fallback
}
