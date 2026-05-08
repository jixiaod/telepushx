package controller

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"telepushx/common"
	"telepushx/model"
)

func PreviewMessage(c *gin.Context) {
	activeID, ok := parseIntPathParam(c, "id", "Invalid ID format")
	if !ok {
		return
	}

	activity, err := model.GetActiveContentByID(activeID, false)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Error getting active content")
		return
	}

	buttons, err := model.GetButtonsByActivityId(activeID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Error getting active buttons")
		return
	}

	userID, ok := parseIntPathParam(c, "uid", "Invalid UID format")
	if !ok {
		return
	}

	user, err := model.GetUserById(userID, false)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Error getting user")
		return
	}

	bot, err := newTelegramBot()
	if err != nil {
		common.SysError(fmt.Sprintf("Error creating bot: %v", err))
		respondError(c, http.StatusInternalServerError, "Error creating bot")
		return
	}

	user.Name = "预览用户"
	err = sendTelegramMessage(bot, user, activity, buttons)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	respondSuccessWithMessage(c, "Message sent successfully", nil)
}
