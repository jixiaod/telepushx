package controller

import (
	"net/http"
	"telepushx/model"

	"github.com/gin-gonic/gin"
)

func SetUserStatus(c *gin.Context) {
	intUid, ok := parseIntPathParam(c, "uid", "Invalid UID format")
	if !ok {
		return
	}

	err := model.UpdateUserStatusById(intUid, 0)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	respondSuccess(c, nil)
}
