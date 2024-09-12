package main

import (
	"os"
	"strconv"
	"time"

	"telepushx/common"
	"telepushx/model"

	"telepushx/router"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	// Assuming MySQL, adjust if using a different database
)

const (
	telegramAPIBaseURL = "https://api.telegram.org/bot"
	botToken           = "6253545273:AAEzdFppjluWM_QplxoZiMsi0GJzDgNmEVI"                           // Replace with your actual bot token
	dbConnectionString = "xiaozhushou_tiger:dxFNKCfddzFDaXr2@tcp(localhost:3306)/xiaozhushou_tiger" // Replace with your actual database connection string
	maxSendRate        = 30                                                                         // Maximum messages per second
	maxSendDuration    = 20 * time.Minute                                                           // Maximum duration for sending messages
	appImageBaseURL    = "https://tiger.ytxzs.com"
)

type PushRequest struct {
	ID int `json:"id"`
}

// Import the mysql package

func main() {

	err := godotenv.Load(".env")
	if err != nil {
		common.FatalLog(err)
	}

	common.SetupGinLog()
	common.SysLog("TelepushX " + common.Version + " started")
	if os.Getenv("GIN_MODE") != "debug" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Connect to the database
	err = model.InitDB()
	if err != nil {
		common.FatalLog(err)
	}
	defer func() {
		err := model.CloseDB()
		if err != nil {
			common.FatalLog(err)
		}
	}()

	// Initialize Redis
	err = common.InitRedisClient()
	if err != nil {
		common.FatalLog(err)
	}

	server := gin.Default()
	router.SetRouter(server)
	var port = os.Getenv("PORT")
	if port == "" {
		port = strconv.Itoa(*common.Port)
	}
	err = server.Run(":" + port)
	if err != nil {
		common.FatalLog(err)
	}
}
