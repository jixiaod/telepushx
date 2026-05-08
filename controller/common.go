package controller

import (
	"net/http"
	"telepushx/common"
	"telepushx/model"

	"github.com/gin-gonic/gin"
)

func GetStatus(c *gin.Context) {
	respondSuccess(c, gin.H{
		"version":        common.Version,
		"start_time":     common.StartTime,
		"system_name":    common.SystemName,
		"server_address": common.ServerAddress,
	})
}

func GetActiveUserCount(c *gin.Context) {
	count, err := model.GetActiveUserCount()
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	respondSuccess(c, gin.H{"count": count})
}
