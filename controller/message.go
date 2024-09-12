package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"sync"
	"telepushx/common"
	"telepushx/model"

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
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid ID format",
			"data":    gin.H{},
		})
		return
	}

	activity, err := model.GetActiveContentByID(intId, false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Error getting active content",
			"data":    gin.H{},
		})
		return
	}

	go doPushMessage(activity)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Push process started",
		"data":    gin.H{},
	})
	//return nil
}

func doPushMessage(activity *model.Activity) {

	users, err := model.GetAllUsers(0, common.GetAllUsersLimitSizeNum)
	if err != nil {
		common.SysError(fmt.Sprintf("Error getting users: %v", err))
		return
	}

	bot, err := tgbotapi.NewBotAPI(os.Getenv("TELEGRAM_BOT_TOKEN"))
	if err != nil {
		common.SysError(fmt.Sprintf("Error creating bot: %v", err))
		return
	}

	stats := common.NewPushStats(len(users))
	limiter := rate.NewLimiter(rate.Limit(30), 1)
	ctx, cancel := context.WithTimeout(context.Background(), common.PushJobStopDuration)
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
					common.SysLog(fmt.Sprintf("Rate limit exceeded for user %s adding back to the front of the queue", u.ChatId))
					return
				}

				err = sendTelegramMessage(bot, u, activity)

				if err != nil {
					common.SysLog(fmt.Sprintf("Error sending message to user %s: %v", u.ChatId, err))
					stats.IncrementFailed()
				} else {
					common.SysLog(fmt.Sprintf("Message sent successfully to user %s", u.ChatId))
					stats.IncrementSuccess()
				}
			}
		}(user)
	}

	wg.Wait()
	common.SysLog("Push process completed or timed out")
	successRate := stats.GetSuccessRate()
	common.SysLog(fmt.Sprintf("Push process %d:%s completed. Total users: %d, Success rate: %.2f%%", activity.Id, activity.ShopId, stats.TotalUsers, successRate))
}

func PreviewMessage(c *gin.Context) {
	uid := c.Param("uid")
	id := c.Param("id")

	// Convert id from string to int
	activeID, err := strconv.Atoi(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid ID format",
			"data":    gin.H{},
		})
		return
	}

	activity, err := model.GetActiveContentByID(activeID, false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Error getting active content",
			"data":    gin.H{},
		})
		return
	}

	// Convert id from string to int
	userID, err := strconv.Atoi(uid)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid UID format",
			"data":    gin.H{},
		})
		return
	}
	user, err := model.GetUserById(userID, false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Error getting user",
			"data":    gin.H{},
		})
		return
	}

	bot, err := tgbotapi.NewBotAPI(os.Getenv("TELEGRAM_BOT_TOKEN"))
	if err != nil {
		common.SysError(fmt.Sprintf("Error creating bot: %v", err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Error creating bot",
			"data":    gin.H{},
		})
		return
	}

	err = sendTelegramMessage(bot, user, activity)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Error sending message",
			"data":    gin.H{},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Message sent successfully",
		"data":    gin.H{},
	})
}

func sendTelegramMessage(bot *tgbotapi.BotAPI, u *model.User, activity *model.Activity) (err error) {
	chatID, err := strconv.ParseInt(u.ChatId, 10, 64)
	if err != nil {
		common.SysError(fmt.Sprintf("Error parsing image JSON for user %s: %v", u.ChatId, err))
		return
	}
	var images []string
	err = json.Unmarshal([]byte(activity.Image), &images)
	if err != nil {
		common.SysError(fmt.Sprintf("Error parsing image JSON for user %s: %v", u.ChatId, err))
		return
	}

	if len(images) > 0 && activity.Type == 0 {
		photo := tgbotapi.NewPhoto(chatID, tgbotapi.FileURL(os.Getenv("APP_IMAGE_BASE_URL")+"/uploads/"+images[0]))
		photo.Caption = activity.Content
		_, err = bot.Send(photo)
	} else if activity.Type == 1 {
		video := tgbotapi.NewVideo(chatID, tgbotapi.FileURL(os.Getenv("APP_IMAGE_BASE_URL")+"/uploads/"+activity.Video))
		video.Caption = activity.Content
		_, err = bot.Send(video)
		if err != nil {
			common.SysLog(fmt.Sprintf("Error sending video message to user %s: %v", u.ChatId, err))
			return
		}
		msg := tgbotapi.NewMessage(chatID, activity.Content)
		_, err = bot.Send(msg)
	}
	return err
}
