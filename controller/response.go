package controller

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func respondSuccess(c *gin.Context, data gin.H) {
	respondSuccessWithMessage(c, "", data)
}

func respondSuccessWithMessage(c *gin.Context, message string, data gin.H) {
	if data == nil {
		data = gin.H{}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": message,
		"data":    data,
	})
}

func respondError(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{
		"success": false,
		"message": message,
		"data":    gin.H{},
	})
}

func parseIntPathParam(c *gin.Context, key string, invalidMessage string) (int, bool) {
	value, err := strconv.Atoi(c.Param(key))
	if err != nil {
		respondError(c, http.StatusBadRequest, invalidMessage)
		return 0, false
	}

	return value, true
}
