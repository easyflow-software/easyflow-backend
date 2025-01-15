package main

import (
	"easyflow-backend/pkg/common"
	"easyflow-backend/pkg/database"
	"os"
	"time"
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
}
