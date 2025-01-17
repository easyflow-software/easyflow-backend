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
)

func main() {
	var log = logger.NewLogger(os.Stdout, "WebSocket", "DEBUG", "System")
	var cfg = config.LoadDefaultConfig()

	var isConnected = false
	var dbInst *database.DatabaseInst
	var connectionAttempts = 0
	var connectionPause = 5
	for !isConnected {
		var err error
		dbInst, err = database.NewDatabaseInst(cfg.DatabaseURL, &cfg.GormConfig)

		if err != nil {
			if connectionAttempts <= 5 {
				connectionAttempts++
				log.PrintfError("Failed to connect to database, retrying in %d seconds. Attempt %d", connectionPause, connectionAttempts)
				time.Sleep(time.Duration(connectionPause) * time.Second)
				connectionPause += 5
			} else {
				panic(err)
			}
		} else {
			isConnected = true
		}
	}

	if err := dbInst.Migrate(); err != nil {
		panic(err)
	}

	var hub = socket.NewHub(dbInst.GetClient(), log)
	go hub.Run()
	// Register the WebSocket handler
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				// Log the error
				log.PrintfError("Panic recovered: %v", err)
				// Return an internal server error to the client
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()
		token, err := r.Cookie("access_token")
		if err != nil {
			log.PrintfWarning("Failed to get access token from cookie")
			http.Error(w, "Failed to get access token from cookie", http.StatusBadRequest)
			return
		}

		payload, err := jwt.ValidateToken(cfg.JwtSecret, token.Value)
		if err != nil {
			log.PrintfError("Failed to validate token")
			http.Error(w, "Failed to validate token", http.StatusUnauthorized)
			return
		}
		socket.ServeWs(hub, payload, w, r)
	})

	// Start the server on port 4000
	log.Printf("WebSocket server starting on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.PrintfError("ListenAndServe: %s", err)
	}
}
