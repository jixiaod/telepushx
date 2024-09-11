package controller

import (
	"net/http"
	"telepushx/common"

	"github.com/gin-gonic/gin"
)

func GetStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"version":        common.Version,
			"start_time":     common.StartTime,
			"system_name":    common.SystemName,
			"server_address": common.ServerAddress,
		},
	})
	return
}
