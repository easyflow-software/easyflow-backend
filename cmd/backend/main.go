package main

import (
	"easyflow-backend/pkg/api/middleware"
	"easyflow-backend/pkg/api/routes/auth" // Authentication route handlers
	"easyflow-backend/pkg/api/routes/chat" // Chat functionality route handlers
	"easyflow-backend/pkg/api/routes/user" // User management route handlers
	"easyflow-backend/pkg/config"          // Application configuration
	"easyflow-backend/pkg/database"        // Database connection and operations
	"easyflow-backend/pkg/logger"          // Custom logging implementation
	"easyflow-backend/pkg/retry"
	"os"
	"strings"
	"time"

	cors "github.com/OnlyNico43/gin-cors" // CORS middleware
	"github.com/gin-gonic/gin"            // Web framework
	"github.com/valkey-io/valkey-go"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger" // Database logging
)

func main() {
	// Load application configuration with default values
	cfg := config.LoadDefaultConfig()

	// Initialize logger for the main package
	log := logger.NewLogger(os.Stdout, "Main", cfg.LogLevel, "System")

	var logLevel gormLogger.LogLevel
	// Configure application mode and database logging based on debug setting
	if !cfg.DebugMode {
		log.PrintfInfo("Starting in release mode")
		gin.SetMode(gin.ReleaseMode)
		logLevel = gormLogger.Silent
	} else {
		log.PrintfInfo("Starting in debug mode")
		gin.SetMode(gin.DebugMode)
		logLevel = gormLogger.Info
	}

	var err error

	connectToDatabase := retry.WithRetry(func() (*database.DatabaseInst, error) {
		return database.NewDatabaseInst(cfg.DatabaseURL, &gorm.Config{
			Logger: gormLogger.Default.LogMode(logLevel),
		})
	}, log, nil)

	dbInst, err := connectToDatabase()
	if err != nil {
		log.PrintfError("Failed to connect to database: %s", err)
		panic(err)
	}

	// Run database migrations
	err = dbInst.Migrate()
	if err != nil {
		panic(err)
	}

	// Set up valkey connection
	connectValkeyClient := retry.WithRetry(func() (valkey.Client, error) {
		return valkey.NewClient(valkey.ClientOption{
			Username:    cfg.ValkeyUsername,
			Password:    cfg.ValkeyPassword,
			ClientName:  cfg.ValkeyClientName,
			InitAddress: []string{cfg.ValkeyURL},
		})
	}, log, nil)

	valkeyClient, err := connectValkeyClient()

	if err != nil {
		log.PrintfError("Failed to connect to Valkey: %s", err)
		panic(err)
	}

	// Initialize Gin router with default middleware
	router := gin.New()

	// Configure trusted proxies for security
	err = router.SetTrustedProxies(nil)
	if err != nil {
		log.PrintfError("Could not set trusted proxies list")
		return
	}

	// Configure router path handling
	router.RedirectFixedPath = true     // Redirect to the correct path if case-insensitive match found
	router.RedirectTrailingSlash = true // Automatically handle trailing slashes

	// Set up CORS middleware
	log.PrintfInfo("Frontend URL for cors: %s", cfg.FrontendURL)
	router.Use(cors.CorsMiddleware(cors.Config{
		AllowedOrigins:   strings.Split(cfg.FrontendURL, ", "),
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE"},
		AllowedHeaders:   []string{"Authorization", "Content-Length", "Content-Type"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Add middleware for database access, configuration, and panic recovery
	router.Use(middleware.DatabaseMiddleware(dbInst.GetClient()))
	router.Use(middleware.ConfigMiddleware(cfg))
	router.Use(middleware.ValkeyMiddleware(valkeyClient))
	router.Use(gin.Recovery())

	// Register API endpoints by feature group
	userEndpoints := router.Group("/user")
	{
		log.PrintfInfo("Registering user endpoints")
		user.RegisterUserEndpoints(userEndpoints)
	}

	authEndpoints := router.Group("/auth")
	{
		log.PrintfInfo("Registering auth endpoints")
		auth.RegisterAuthEndpoints(authEndpoints)
	}

	chatEndpoints := router.Group("/chat")
	{
		log.PrintfInfo("Registering chat endpoints")
		chat.RegisterChatEndpoints(chatEndpoints)
	}

	// Start the HTTP server
	log.PrintfInfo("Starting server on port %s", cfg.Port)
	if err := router.Run(":" + cfg.Port); err != nil {
		log.PrintfError("Failed to start server: %s", err)
		return
	}
}
