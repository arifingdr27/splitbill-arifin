package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

const (
	StorageVM       = "VM"
	StorageFirebase = "FIREBASE"
)

type Config struct {
	AppPort        string
	AppEnv         string
	CORSOrigins    string
	GeminiAPIKey   string
	GeminiModel    string
	BucketStorage  string
	StorageLocalPath string
	FirebaseCredPath string
	FirebaseBucket   string
	LogLevel         string
	LogDir           string
}

func Load(envFiles ...string) (*Config, error) {
	_ = godotenv.Load(envFiles...)

	cfg := &Config{
		AppPort:          getEnv("APP_PORT", "3000"),
		AppEnv:           getEnv("APP_ENV", "development"),
		CORSOrigins:      getEnv("CORS_ALLOW_ORIGINS", "*"),
		GeminiAPIKey:     os.Getenv("GEMINI_API_KEY"),
		GeminiModel:      getEnv("GEMINI_MODEL", "gemini-2.0-flash"),
		BucketStorage:    strings.ToUpper(getEnv("BUCKET_STORAGE", StorageVM)),
		StorageLocalPath: getEnv("STORAGE_LOCAL_PATH", "./storage/public"),
		FirebaseCredPath: getEnv("FIREBASE_SERVICE_ACCOUNT_KEY_PATH", "./storage/firebase-adminsdk.json"),
		FirebaseBucket:   os.Getenv("FIREBASE_STORAGE_BUCKET"),
		LogLevel:         getEnv("LOG_LEVEL", "info"),
		LogDir:           getEnv("LOG_DIR", "./storage/logs"),
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
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
	return nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
