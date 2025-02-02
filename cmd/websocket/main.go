package main

import (
	"net/http"
	"os"
	"time"

	"easyflow-backend/pkg/config"
	"easyflow-backend/pkg/database"
	"easyflow-backend/pkg/jwt"
	"easyflow-backend/pkg/logger"
	"easyflow-backend/pkg/socket"

	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
)

func main() {
	// Initialize logger specifically for WebSocket operations
	var log = logger.NewLogger(os.Stdout, "WebSocket", "DEBUG", "System")

	// Load application configuration
	var cfg = config.LoadDefaultConfig()

	var logLevel gormLogger.LogLevel
	// Configure application mode and database logging based on debug setting
	if !cfg.DebugMode {
		log.PrintfInfo("Starting in release mode")
		logLevel = gormLogger.Silent
	} else {
		log.PrintfInfo("Starting in debug mode")
		logLevel = gormLogger.Info
	}
	// Database connection retry logic
	var isConnected = false
	var dbInst *database.DatabaseInst
	var connectionAttempts = 0
	var connectionPause = 5 // Initial pause duration in seconds

	// Attempt database connection with exponential backoff
	for !isConnected {
		var err error
		dbInst, err = database.NewDatabaseInst(cfg.DatabaseURL, &gorm.Config{Logger: gormLogger.Default.LogMode(logLevel)})
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

	// Run database migrations
	if err := dbInst.Migrate(); err != nil {
		panic(err)
	}

	// Initialize WebSocket hub for managing connections
	var hub = socket.NewHub(dbInst.GetClient(), log)
	// Start the hub in a separate goroutine
	go hub.Run()

	// Register the WebSocket handler for the root path
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Recover from panics to prevent server crashes
		defer func() {
			if err := recover(); err != nil {
				// Log the recovered panic
				log.PrintfError("Panic recovered: %v", err)
				// Return 500 error to client
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()

		// Extract JWT token from cookies
		token, err := r.Cookie("access_token")
		if err != nil {
			log.PrintfWarning("Failed to get access token from cookie")
			http.Error(w, "Failed to get access token from cookie", http.StatusBadRequest)
			return
		}

		// Validate the JWT token
		payload, err := jwt.ValidateToken(cfg.JwtSecret, token.Value)
		if err != nil {
			log.PrintfError("Failed to validate token")
			http.Error(w, "Failed to validate token", http.StatusUnauthorized)
			return
		}

		// Upgrade HTTP connection to WebSocket connection
		socket.ServeWs(hub, payload, w, r)
	})

	// Start the WebSocket server
	log.PrintfInfo("WebSocket server starting on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.PrintfError("ListenAndServe: %s", err)
	}
}
