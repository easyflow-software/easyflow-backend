package main

import (
	"easyflow-backend/pkg/api/middleware"
	"easyflow-backend/pkg/api/routes/auth"
	"easyflow-backend/pkg/api/routes/chat"
	"easyflow-backend/pkg/api/routes/user"
	"easyflow-backend/pkg/config"
	"easyflow-backend/pkg/database"
	"easyflow-backend/pkg/logger"

	"os"
	"strings"
	"time"

	cors "github.com/OnlyNico43/gin-cors"
	"github.com/gin-gonic/gin"
	gormLogger "gorm.io/gorm/logger"
)

func main() {
	cfg := config.LoadDefaultConfig()

	log := logger.NewLogger(os.Stdout, "Main", cfg.LogLevel, "System")
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

	if !cfg.DebugMode {
		gin.SetMode(gin.ReleaseMode)
		dbInst.SetLogMode(gormLogger.Silent)
	}

	err := dbInst.Migrate()
	if err != nil {
		panic(err)
	}

	router := gin.New()

	err = router.SetTrustedProxies(nil)
	if err != nil {
		log.PrintfError("Could not set trusted proxies list")
		return
	}

	router.RedirectFixedPath = true
	router.RedirectTrailingSlash = true

	log.Printf("Frontend URL for cors: %s", cfg.FrontendURL)

	router.Use(cors.CorsMiddleware(cors.Config{
		AllowedOrigins:   strings.Split(cfg.FrontendURL, ", "),
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE"},
		AllowedHeaders:   []string{"Authorization", "Content-Length", "Content-Type"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	router.Use(middleware.DatabaseMiddleware(dbInst.GetClient()))
	router.Use(middleware.ConfigMiddleware(cfg))
	router.Use(gin.Recovery())

	//register user endpoints
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

	log.Printf("Starting server on port %s", cfg.Port)
	if err := router.Run(":" + cfg.Port); err != nil {
		log.PrintfError("Failed to start server: %s", err)
		return
	}
}
