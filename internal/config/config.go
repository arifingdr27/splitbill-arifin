package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

const (
	StorageVM       = "VM"
	StorageFirebase = "FIREBASE"

	DefaultBodyLimitBytes     = 5 * 1024 * 1024
	DefaultReadTimeoutSec     = 30
	DefaultWriteTimeoutSec    = 60
	DefaultIdleTimeoutSec     = 60
	DefaultHTTPConcurrency    = 128
	DefaultRateLimitRPM       = 5
	DefaultExtractMaxConcurrent = 5
	DefaultGeminiTimeoutSec     = 25
	DefaultGroqModel            = "llama-3.1-8b-instant"
)

type Config struct {
	AppPort          string
	AppEnv           string
	CORSOrigins      string
	GeminiAPIKey     string
	GeminiModel      string
	BucketStorage    string
	StorageLocalPath string
	FirebaseCredPath string
	FirebaseBucket   string
	LogLevel         string
	LogDir           string

	HTTPBodyLimitBytes   int
	HTTPReadTimeout      time.Duration
	HTTPWriteTimeout     time.Duration
	HTTPIdleTimeout      time.Duration
	HTTPConcurrency      int
	RateLimitRPM         int
	ExtractMaxConcurrent int
	GeminiTimeout        time.Duration
	GroqAPIKey           string
	GroqModel            string
	GroqTimeout          time.Duration
}

func Load(envFiles ...string) (*Config, error) {
	_ = godotenv.Load(envFiles...)

	readSec := getEnvInt("HTTP_READ_TIMEOUT_SEC", DefaultReadTimeoutSec)
	writeSec := getEnvInt("HTTP_WRITE_TIMEOUT_SEC", DefaultWriteTimeoutSec)
	idleSec := getEnvInt("HTTP_IDLE_TIMEOUT_SEC", DefaultIdleTimeoutSec)
	geminiSec := getEnvInt("GEMINI_TIMEOUT_SEC", DefaultGeminiTimeoutSec)
	groqSec := getEnvInt("GROQ_TIMEOUT_SEC", geminiSec)

	cfg := &Config{
		AppPort:          getEnv("APP_PORT", "3000"),
		AppEnv:           getEnv("APP_ENV", "development"),
		CORSOrigins:      getEnv("CORS_ALLOW_ORIGINS", "*"),
		GeminiAPIKey:     os.Getenv("GEMINI_API_KEY"),
		GeminiModel:      getEnv("GEMINI_MODEL", "gemini-3.6-flash"),
		BucketStorage:    strings.ToUpper(getEnv("BUCKET_STORAGE", StorageVM)),
		StorageLocalPath: getEnv("STORAGE_LOCAL_PATH", "./storage/public"),
		FirebaseCredPath: getEnv("FIREBASE_SERVICE_ACCOUNT_KEY_PATH", "./storage/firebase-adminsdk.json"),
		FirebaseBucket:   os.Getenv("FIREBASE_STORAGE_BUCKET"),
		LogLevel:         getEnv("LOG_LEVEL", "info"),
		LogDir:           getEnv("LOG_DIR", "./storage/logs"),

		HTTPBodyLimitBytes:   getEnvInt("HTTP_BODY_LIMIT_BYTES", DefaultBodyLimitBytes),
		HTTPReadTimeout:      time.Duration(readSec) * time.Second,
		HTTPWriteTimeout:     time.Duration(writeSec) * time.Second,
		HTTPIdleTimeout:      time.Duration(idleSec) * time.Second,
		HTTPConcurrency:      getEnvInt("HTTP_CONCURRENCY", DefaultHTTPConcurrency),
		RateLimitRPM:         getEnvInt("RATE_LIMIT_RPM", DefaultRateLimitRPM),
		ExtractMaxConcurrent: getEnvInt("EXTRACT_MAX_CONCURRENT", DefaultExtractMaxConcurrent),
		GeminiTimeout:        time.Duration(geminiSec) * time.Second,
		GroqAPIKey:           os.Getenv("GROQ_API_KEY"),
		GroqModel:            getEnv("GROQ_MODEL", DefaultGroqModel),
		GroqTimeout:          time.Duration(groqSec) * time.Second,
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) IsProduction() bool {
	return strings.EqualFold(c.AppEnv, "production")
}

func (c *Config) validate() error {
	if c.GeminiAPIKey == "" {
		return fmt.Errorf("GEMINI_API_KEY is required")
	}
	switch c.BucketStorage {
	case StorageVM, StorageFirebase:
	default:
		return fmt.Errorf("BUCKET_STORAGE must be VM or FIREBASE, got %q", c.BucketStorage)
	}
	if c.BucketStorage == StorageFirebase && c.FirebaseBucket == "" {
		return fmt.Errorf("FIREBASE_STORAGE_BUCKET is required when BUCKET_STORAGE=FIREBASE")
	}
	if c.IsProduction() && (c.CORSOrigins == "" || c.CORSOrigins == "*") {
		return fmt.Errorf("CORS_ALLOW_ORIGINS must be set to exact frontend origin(s) when APP_ENV=production")
	}
	if c.HTTPBodyLimitBytes < 1 {
		return fmt.Errorf("HTTP_BODY_LIMIT_BYTES must be > 0")
	}
	if c.RateLimitRPM < 1 {
		return fmt.Errorf("RATE_LIMIT_RPM must be > 0")
	}
	if c.ExtractMaxConcurrent < 1 {
		return fmt.Errorf("EXTRACT_MAX_CONCURRENT must be > 0")
	}
	if c.GeminiTimeout < time.Second {
		return fmt.Errorf("GEMINI_TIMEOUT_SEC must be >= 1")
	}
	if c.GroqAPIKey != "" && c.GroqTimeout < time.Second {
		return fmt.Errorf("GROQ_TIMEOUT_SEC must be >= 1")
	}
	return nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}
