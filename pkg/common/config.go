package common

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Config struct {
	// stage
	Stage string
	// log level
	LogLevel LogLevel
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
	// ws
	WebsocketPort string
	// s3
	BucketURL                string
	BucketAccessKeyId        string
	BucketSecret             string
	ProfilePictureBucketName string
	// app
	FrontendURL string
	Domain      string
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

// LoadDefaultConfig loads the default configuration values.
// It reads the environment variables from the .env file, if present,
// and returns a Config struct with the loaded values.
func LoadDefaultConfig() *Config {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file: ", err)
	}

	return &Config{
		GormConfig: gorm.Config{
			Logger:                                   logger.Default.LogMode(logger.Silent),
			DisableForeignKeyConstraintWhenMigrating: true,
		},
		Stage:                    getEnv("STAGE", "development"),
		LogLevel:                 LogLevel(getEnv("LOG_LEVEL", "DEBUG")),
		DatabaseURL:              getEnv("DATABASE_URL", ""),
		SaltRounds:               getEnvInt("SALT_OR_ROUNDS", 10),
		JwtSecret:                getEnv("JWT_SECRET", "public_secret"),
		JwtExpirationTime:        getEnvInt("JWT_EXPIRATION_TIME", 60*10),          // 10 minutes
		RefreshExpirationTime:    getEnvInt("REFRESH_EXPIRATION_TIME", 60*60*24*7), // 1 week
		Port:                     getEnv("PORT", "4000"),
		DebugMode:                getEnv("DEBUG_MODE", "false") == "true",
		WebsocketPort:            getEnv("WEBSOCKET_PORT", "4001"),
		BucketURL:                getEnv("BUCKET_URL", ""),
		BucketAccessKeyId:        getEnv("BUCKET_ACCESS_KEY_ID", ""),
		BucketSecret:             getEnv("BUCKET_SECRET", ""),
		ProfilePictureBucketName: getEnv("PROFILE_PICTURE_BUCKET_NAME", ""),
		FrontendURL:              getEnv("FRONTEND_URL", "http://localhost:3000"),
		Domain:                   getEnv("DOMAIN", "localhost"),
	}
}
