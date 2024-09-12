package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"sync"
	"telepushx/common"
	"telepushx/model"
	"time"

	"github.com/gin-gonic/gin"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"golang.org/x/time/rate"
)

func PushMessage(c *gin.Context) {

	//user := model.User{ID: c.Param("id")}
	id := c.Param("id")

	// Convert id from string to int
	intId, err := strconv.Atoi(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	activity, err := model.GetActiveContentByID(intId, false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error getting active content"})
		return
	}

	go doPushMessage(activity)

	c.JSON(http.StatusOK, gin.H{"message": "Push process started"})
	//return nil
}

func doPushMessage(activity *model.Activity) {

	users, err := model.GetAllUsers(0, common.GetAllUsersLimitSizeNum)
	if err != nil {
		common.FatalLog("Error getting users: %v", err)
		return
	}

	bot, err := tgbotapi.NewBotAPI(os.Getenv("TELEGRAM_BOT_TOKEN"))
	if err != nil {
		common.FatalLog("Error creating bot: %v", err)
		return
	}

	limiter := rate.NewLimiter(rate.Limit(30), 1)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	for _, user := range users {
		wg.Add(1)
		go func(u *model.User) {
			defer wg.Done()

			select {
			case <-ctx.Done():
				return
			default:
				if err := limiter.Wait(ctx); err != nil {
					// If rate limit is exceeded, add the user back to the front of the queue
					users = append([]*model.User{u}, users...)
					common.SysLog("Rate limit exceeded for user " + u.ChatId + " adding back to the front of the queue")
					return
				}

				chatID, err := strconv.ParseInt(u.ChatId, 10, 64)
				if err != nil {
					common.FatalLog("Error parsing ChatID for user %s: %v", u.ChatId, err)
					return
				}

				var images []string
				err = json.Unmarshal([]byte(activity.Image), &images)
				if err != nil {
					common.FatalLog("Error parsing image JSON for user %s: %v", u.ChatId, err)
					return
				}

				if len(images) > 0 {
					photo := tgbotapi.NewPhoto(chatID, tgbotapi.FileURL(os.Getenv("APP_IMAGE_BASE_URL")+"/uploads/"+images[0]))
					photo.Caption = activity.Content
					_, err = bot.Send(photo)
				} else {
					msg := tgbotapi.NewMessage(chatID, activity.Content)
					_, err = bot.Send(msg)
				}

				if err != nil {
					common.SysLog("Error sending message to user " + u.ChatId + " " + err.Error())
				} else {
					common.SysLog("Message sent successfully to user " + u.ChatId)
				}
			}
		}(user)
	}

	wg.Wait()
	common.SysLog("Push process completed or timed out")
}
