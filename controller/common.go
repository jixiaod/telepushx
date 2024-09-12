package controller

import (
	"net/http"
	"telepushx/common"
	"telepushx/model"

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

}

func GetActiveUserCount(c *gin.Context) {
	count, err := model.GetActiveUserCount()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": err.Error(),
			"data":    gin.H{},
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    gin.H{"count": count},
	})
}
