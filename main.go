package main

import (
	"os"
	"strconv"
	"telepushx/common"
	"telepushx/model"
	"telepushx/router"
	"telepushx/task"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(".env"); err != nil {
		common.FatalLog(err)
	}

	initializeApplication()
	defer closeDatabase()
	startBackgroundTasks()

	if err := newServer().Run(":" + resolvePort()); err != nil {
		common.FatalLog(err)
	}
}

func initializeApplication() {
	common.Init()
	common.SetupDailyRotateLog()
	common.SysLog("TelepushX " + common.Version + " started")

	if os.Getenv("GIN_MODE") != "debug" {
		gin.SetMode(gin.ReleaseMode)
	}

	if err := model.InitDB(); err != nil {
		common.FatalLog(err)
	}

	if err := common.InitRedisClient(); err != nil {
		common.FatalLog(err)
	}
}

func closeDatabase() {
	if err := model.CloseDB(); err != nil {
		common.FatalLog(err)
	}
}

func startBackgroundTasks() {
	nextMinute := time.Now().Truncate(time.Minute).Add(time.Minute)
	time.Sleep(time.Until(nextMinute))
	task.StartPushChecker()
}

func newServer() *gin.Engine {
	server := gin.Default()
	router.SetRouter(server)
	return server
}

func resolvePort() string {
	port := os.Getenv("PORT")
	if port != "" {
		return port
	}

	return strconv.Itoa(*common.Port)
}
