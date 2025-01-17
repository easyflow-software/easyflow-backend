package config

import (
	"easyflow-backend/pkg/logger"

	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"gorm.io/gorm"
)

type Config struct {
	// stage
	Stage string
	// log level
	LogLevel logger.LogLevel
	//gorm
	GormConfig gorm.Config
	//env
	DatabaseURL string
	SaltRounds  int
	Port        string
	DebugMode   bool
	//jwt
	JwtSecret             string
	JwtExpirationTime     int
	RefreshExpirationTime int
	// Cookie
	CookieSecret string
	// s3
	BucketURL                string
	BucketAccessKeyId        string
	BucketSecret             string
	ProfilePictureBucketName string
	// app
	FrontendURL string
	Domain      string
	// Cloudflare Turnstile
	TurnstileUrl    string
	TurnstileSecret string
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if value, ok := os.LookupEnv(key); ok {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return fallback
}

// Loads the default configuration values.
// It reads the environment variables from the .env file, if present,
// and returns a Config struct with the loaded values.
func LoadDefaultConfig() *Config {
	err := godotenv.Load("../../.env")
	if err != nil {
		fmt.Println("Error loading .env file: ", err)
	}

	return &Config{
		// Application
		Stage:        getEnv("STAGE", "development"),
		LogLevel:     logger.LogLevel(getEnv("LOG_LEVEL", "DEBUG")),
		Port:         getEnv("PORT", "4000"),
		DebugMode:    getEnv("DEBUG_MODE", "false") == "true",
		FrontendURL:  getEnv("FRONTEND_URL", "http://localhost:3000"),
		Domain:       getEnv("DOMAIN", "localhost"),
		CookieSecret: getEnv("COOKIE_SECRET", "cookie_secret"),
		//Database
		DatabaseURL: getEnv("DATABASE_URL", ""),
		// Crypto
		SaltRounds: getEnvInt("SALT_OR_ROUNDS", 10),
		// JWT
		JwtSecret:             getEnv("JWT_SECRET", "public_secret"),
		JwtExpirationTime:     getEnvInt("JWT_EXPIRATION_TIME", 60*10),          // 10 minutes
		RefreshExpirationTime: getEnvInt("REFRESH_EXPIRATION_TIME", 60*60*24*7), // 1 week
		// Minio
		BucketURL:                getEnv("BUCKET_URL", ""),
		BucketAccessKeyId:        getEnv("BUCKET_ACCESS_KEY_ID", ""),
		BucketSecret:             getEnv("BUCKET_SECRET", ""),
		ProfilePictureBucketName: getEnv("PROFILE_PICTURE_BUCKET_NAME", ""),
		// Turnstile
		TurnstileUrl:    getEnv("TURNSTILE_URL", "https://challenges.cloudflare.com/turnstile/v0/siteverify"),
		TurnstileSecret: getEnv("TURNSTILE_SECRET", ""),
	}
}
