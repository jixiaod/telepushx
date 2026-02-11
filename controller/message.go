package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"telepushx/common"
	"telepushx/model"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"golang.org/x/time/rate"
)

// PushMessageByJob 根据活动id和目标地区推送消息
func PushMessageByJob(id int, targetRegionId int) {
	activity, err := model.GetActiveContentByID(id, false)
	if err != nil {
		common.SysError(fmt.Sprintf("Error getting activity %d: %v", id, err))
		return
	}

	buttons, err := model.GetButtonsByActivityId(id)
	if err != nil {
		common.SysError(fmt.Sprintf("Error getting buttons for activity %d: %v", id, err))
		return
	}

	go doPushMessage(activity, buttons, targetRegionId)
}

func doPushMessage(activity *model.Activity, buttons []*model.Button, targetRegionId int) {
	users, err := model.GetAllUsersWithRegionId(targetRegionId, 0, common.GetAllUsersLimitSizeNum)
	if err != nil {
		common.SysError(fmt.Sprintf("Error getting users: %v", err))
		return
	}

	if len(users) == 0 {
		common.SysLog(fmt.Sprintf("No users in region %d for activity %d", targetRegionId, activity.Id))
		return
	}

	common.SysLog(fmt.Sprintf(
		"Start pushing activity %d to %d users (activityRegion: %d, targetRegion: %d)",
		activity.Id, len(users), activity.RegionId, targetRegionId,
	))

	bot, err := tgbotapi.NewBotAPI(os.Getenv("TELEGRAM_BOT_TOKEN"))
	if err != nil {
		common.SysError(fmt.Sprintf("Error creating bot: %v", err))
		return
	}

	stats := common.NewPushStats(len(users))
	stats.RecordStartTime()

	limiter := rate.NewLimiter(rate.Limit(common.PushJobRateLimitNum), 1)
	if activity.IsPin == 1 {
		limiter = rate.NewLimiter(rate.Limit(common.PinPushJobRateLimitNum), 1)
	}

	d := calculatePushJobStopDuration(activity) - 30*time.Second
	if d < 5*time.Second {
		d = 5 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), d)
	defer cancel()

	var wg sync.WaitGroup
	queue := &model.UserQueue{}
	queue.PushBatch(users)

	maxWorkers := 50
	sem := make(chan struct{}, maxWorkers)

dispatchLoop:
	for {
		user := queue.Pop()
		if user == nil {
			break
		}

		select {
		case <-ctx.Done():
			break dispatchLoop
		default:
		}

		sem <- struct{}{}
		wg.Add(1)

		go func(u *model.User) {
			defer wg.Done()
			defer func() { <-sem }()

			defer func() {
				if r := recover(); r != nil {
					common.SysError(fmt.Sprintf(
						"panic in push goroutine activity=%d user=%s r=%v",
						activity.Id, u.ChatId, r,
					))
					common.SysError(string(debug.Stack()))
				}
			}()

			if ctx.Err() != nil {
				return
			}

			if err := limiter.Wait(ctx); err != nil {
				queue.PushFront(u)
				return
			}

			sendErr := sendTelegramMessage(bot, u, activity, buttons)
			if sendErr != nil {
				errMsg := sendErr.Error()
				if strings.Contains(errMsg, "Gateway Timeout") || strings.Contains(errMsg, "Too Many Requests") {
					queue.PushFront(u)
					time.Sleep(2 * time.Second)
					return
				}
				if strings.Contains(errMsg, "Forbidden") {
					stats.IncrementFailed()
					model.UpdateUserStatusById(int(u.Id), 0)
					return
				}
				stats.IncrementFailed()
				return
			}

			stats.IncrementSuccess()
		}(user)

		if !queue.HasNext() {
			break
		}
	}

	wg.Wait()
	stats.RecordEndTime()

	common.SysLog(fmt.Sprintf(
		"Push completed activity %d (region %d): Total=%d, Success=%d, Failed=%d",
		activity.Id, targetRegionId, stats.TotalUsers, stats.SuccessfulPush, stats.FailedPush,
	))
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

	buttons, err := model.GetButtonsByActivityId(activeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Error getting active buttons",
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

	user.Name = "预览用户"
	err = sendTelegramMessage(bot, user, activity, buttons)
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
		"message": "Message sent successfully",
		"data":    gin.H{},
	})
}

func sendTelegramMessage(bot *tgbotapi.BotAPI, u *model.User, activity *model.Activity, buttons []*model.Button) error {
	chatID, err := strconv.ParseInt(u.ChatId, 10, 64)
	if err != nil {
		return err
	}

	var images []string
	if err := json.Unmarshal([]byte(activity.Image), &images); err != nil {
		return err
	}

	if len(images) > 0 && activity.Type == 0 {
		photo := tgbotapi.NewPhoto(chatID, tgbotapi.FileURL(os.Getenv("APP_IMAGE_BASE_URL")+"/uploads/"+images[0]))
		photo.Caption = "亲爱的" + common.FilterName(u.Name) + ":\n\n" + common.Text(activity.Content)
		photo.ParseMode = "HTML"
		if len(buttons) > 0 {
			photo.ReplyMarkup = tgbotapi.InlineKeyboardMarkup{
				InlineKeyboard: buildButtonOptions(buttons),
			}
		}
		sentMsg, err := bot.Send(photo)
		if err != nil {
			return err
		}
		if activity.IsPin == 1 {
			_, _ = bot.Request(tgbotapi.PinChatMessageConfig{
				ChatID:    chatID,
				MessageID: sentMsg.MessageID,
			})
			time.Sleep(500 * time.Millisecond)
		}
	} else if activity.Type == 1 {
		video := tgbotapi.NewVideo(chatID, tgbotapi.FileURL(os.Getenv("APP_IMAGE_BASE_URL")+"/uploads/"+activity.Video))
		video.Caption = "亲爱的" + common.FilterName(u.Name) + ":\n\n" + common.Text(activity.Content)
		video.ParseMode = "HTML"
		if len(buttons) > 0 {
			video.ReplyMarkup = tgbotapi.InlineKeyboardMarkup{
				InlineKeyboard: buildButtonOptions(buttons),
			}
		}
		sentMsg, err := bot.Send(video)
		if err != nil {
			return err
		}
		if activity.IsPin == 1 {
			_, _ = bot.Request(tgbotapi.PinChatMessageConfig{
				ChatID:    chatID,
				MessageID: sentMsg.MessageID,
			})
			time.Sleep(500 * time.Millisecond)
		}
	}
	return nil
}

func buildButtonOptions(buttons []*model.Button) [][]tgbotapi.InlineKeyboardButton {
	maxLine := 0
	for _, b := range buttons {
		if b.OneLine > maxLine {
			maxLine = b.OneLine
		}
	}

	var result [][]tgbotapi.InlineKeyboardButton
	for line := 1; line <= maxLine; line++ {
		var row []tgbotapi.InlineKeyboardButton
		for _, b := range buttons {
			if b.OneLine == line {
				row = append(row, buildButton(b))
			}
		}
		if len(row) > 0 {
			result = append(result, row)
		}
	}
	return result
}

func buildButton(button *model.Button) tgbotapi.InlineKeyboardButton {
	if button.Inline != "" {
		return tgbotapi.NewInlineKeyboardButtonData(button.Text, button.Inline)
	} else {
		link := button.Link
		if link == "" {
			link = "https://t.me/Ytxzs_bot"
		}
		return tgbotapi.NewInlineKeyboardButtonURL(button.Text, link)
	}
}

func calculatePushJobStopDuration(activity *model.Activity) time.Duration {
	pushDuration := common.PushJobStopDuration

	if activity != nil && activity.CountTime > 0 {
		return time.Duration(activity.CountTime) * time.Minute
	}

	rows, err := model.GetAllActivitiesOrderByTime()
	if err != nil || len(rows) == 0 {
		return pushDuration
	}

	var times []time.Time
	layout := "15:04:05"
	for _, a := range rows {
		tStr := a.ActivityTime
		if tStr == "" {
			continue
		}
		tm, err := time.Parse(layout, tStr)
		if err != nil {
			continue
		}
		times = append(times, tm.UTC())
	}

	sort.Slice(times, func(i, j int) bool { return times[i].Before(times[j]) })

	currentTime := time.Now()
	currentTime = time.Date(0, 1, 1, currentTime.Hour(), currentTime.Minute(), currentTime.Second(), 0, time.UTC)

	for _, t := range times {
		if currentTime.Before(t) {
			return t.Sub(currentTime)
		}
	}

	return times[0].Add(24 * time.Hour).Sub(currentTime)
}