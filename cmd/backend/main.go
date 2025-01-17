package main

import (
	"easyflow-backend/pkg/api/middleware"
	"easyflow-backend/pkg/api/routes/auth" // Authentication route handlers
	"easyflow-backend/pkg/api/routes/chat" // Chat functionality route handlers
	"easyflow-backend/pkg/api/routes/user" // User management route handlers
	"easyflow-backend/pkg/config"          // Application configuration
	"easyflow-backend/pkg/database"        // Database connection and operations
	"easyflow-backend/pkg/logger"          // Custom logging implementation
	cors "github.com/OnlyNico43/gin-cors"  // CORS middleware
	"github.com/gin-gonic/gin"             // Web framework
	gormLogger "gorm.io/gorm/logger"       // Database logging
	"os"
	"strings"
	"time"
)

func main() {
	// Load application configuration with default values
	cfg := config.LoadDefaultConfig()

	// Initialize logger for the main package
	log := logger.NewLogger(os.Stdout, "Main", cfg.LogLevel, "System")

	// Database connection retry logic
	var isConnected = false
	var dbInst *database.DatabaseInst
	var connectionAttempts = 0
	var connectionPause = 5 // Initial pause duration in seconds

	// Attempt database connection with exponential backoff
	for !isConnected {
		var err error
		dbInst, err = database.NewDatabaseInst(cfg.DatabaseURL, &cfg.GormConfig)
		if err != nil {
			if connectionAttempts <= 5 {
				connectionAttempts++
				log.PrintfError("Failed to connect to database, retrying in %d seconds. Attempt %d", connectionPause, connectionAttempts)
				time.Sleep(time.Duration(connectionPause) * time.Second)
				connectionPause += 5 // Increase pause duration for next attempt
			} else {
				panic(err) // Give up after 5 attempts
			}
		} else {
			isConnected = true
		}
	}

	// Configure application mode and database logging based on debug setting
	if !cfg.DebugMode {
		gin.SetMode(gin.ReleaseMode)
		dbInst.SetLogMode(gormLogger.Silent)
	}

	// Run database migrations
	err := dbInst.Migrate()
	if err != nil {
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
	log.Printf("Frontend URL for cors: %s", cfg.FrontendURL)
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
	router.Use(gin.Recovery())

	// Register API endpoints by feature group
	userEndpoints := router.Group("/user")
	{
		log.Printf("Registering user endpoints")
		user.RegisterUserEndpoints(userEndpoints)
	}

	authEndpoints := router.Group("/auth")
	{
		log.Printf("Registering auth endpoints")
		auth.RegisterAuthEndpoints(authEndpoints)
	}

	chatEndpoints := router.Group("/chat")
	{
		log.Printf("Registering chat endpoints")
		chat.RegisterChatEndpoints(chatEndpoints)
	}

	// Start the HTTP server
	log.Printf("Starting server on port %s", cfg.Port)
	if err := router.Run(":" + cfg.Port); err != nil {
		log.PrintfError("Failed to start server: %s", err)
		return
	}
}
