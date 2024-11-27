package controller

import (
	"net/http"
	"strconv"
	"telepushx/model"

	"github.com/gin-gonic/gin"
)

func SetUserStatus(c *gin.Context) {
	uid := c.Param("uid")
	// Convert id from string to int
	intUid, err := strconv.Atoi(uid)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid ID format",
			"data":    gin.H{},
		})
		return
	}
	err = model.UpdateUserStatusById(intUid, 0)
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
		"data":    gin.H{},
	})
}
