package main

import (
	"os"
	"strconv"

	"telepushx/common"
	"telepushx/model"

	"telepushx/router"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	// Assuming MySQL, adjust if using a different database
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
