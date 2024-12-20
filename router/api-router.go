package router

import (
	"telepushx/controller"
	"telepushx/middleware"

	"github.com/gin-gonic/gin"
)

func SetApiRouter(router *gin.Engine) {
	apiRouter := router.Group("/api")
	apiRouter.Use(middleware.GlobalAPIRateLimit())
	{
		apiRouter.GET("/status", middleware.CriticalRateLimit(), controller.GetStatus)
		apiRouter.GET("/user-count", middleware.CriticalRateLimit(), controller.GetActiveUserCount)
		apiRouter.GET("/update-user-stat/:uid", middleware.CriticalRateLimit(), controller.SetUserStatus)
		apiRouter.POST("/push/:id", middleware.CriticalRateLimit(), controller.PushMessage)
		apiRouter.POST("/time", middleware.CriticalRateLimit(), controller.CalculatePushTime)
		apiRouter.POST("/preview/:id/:uid", middleware.CriticalRateLimit(), controller.PreviewMessage)
	}
}
