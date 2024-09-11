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

		apiRouter.GET("/push/:id", middleware.CriticalRateLimit(), controller.PushMessage)
		//apiRouter.GET("/preview/:chat_id/:id", middleware.CriticalRateLimit(), controller.RegisterClient)

	}
}
