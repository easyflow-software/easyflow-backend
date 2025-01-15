package main

import (
	"easyflow-backend/pkg/common"
	"easyflow-backend/pkg/database"
	"easyflow-backend/pkg/middleware"
	"os"
	"strings"
	"time"

	cors "github.com/OnlyNico43/gin-cors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm/logger"
)

func main() {
	cfg := common.LoadDefaultConfig()

	log := common.NewLogger(os.Stdout, "Main", nil, common.LogLevel(cfg.LogLevel))
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
		dbInst.SetLogMode(logger.Silent)
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

	router.Use(cors.CorsMiddleware(cors.Config{
		AllowedOrigins:   strings.Split(cfg.WebsocketPort, ", "),
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE"},
		AllowedHeaders:   []string{"Authorization", "Content-Length", "Content-Type"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	log.Printf("Frontend URL for cors: %s", cfg.WebsocketPort)

	router.Use(middleware.DatabaseMiddleware(dbInst.GetClient()))
	router.Use(middleware.ConfigMiddleware(cfg))
	router.Use(gin.Recovery())

	log.Printf("Starting server on port %s", cfg.WebsocketPort)
	err = router.Run(":" + cfg.WebsocketPort)
	if err != nil {
		log.PrintfError("Failed to start server: %s", err)
		return
	}
}
