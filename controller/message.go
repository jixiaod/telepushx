package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"runtime/debug"

	"telepushx/common"
	"telepushx/model"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"golang.org/x/time/rate"
	"github.com/gin-gonic/gin"
)

// 全局变量，标识是否有正在进行中的推送
var pushingFlag int32 = 0
var PushMessageLock int32 = 0

func PushMessageByJob(id int, targetRegionId int) {
	activity, err := model.GetActiveContentByID(id, false)
	if err != nil {
		return
	}

	buttons, err := model.GetButtonsByActivityId(id)
	if err != nil {
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

	common.SysLog(fmt.Sprintf(
		"Start pushing to %d users for activity ID %d (activityRegion: %d, targetRegion: %d)",
		len(users), activity.Id, activity.RegionId, targetRegionId,
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

	// 抢占推送锁（保证全局只有一个推送任务）
	if !atomic.CompareAndSwapInt32(&pushingFlag, 0, 1) {
		common.SysLog("已有推送正在执行，跳过")
		return
	}
	defer atomic.StoreInt32(&pushingFlag, 0)

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
				errMessage := sendErr.Error()

				switch {
				case strings.Contains(errMessage, "Gateway Timeout"):
					common.SysLog(fmt.Sprintf("Gateway Timeout %d to user %s: %v", activity.Id, u.ChatId, sendErr))
					queue.PushFront(u)
					time.Sleep(10 * time.Second)
					return
				case strings.Contains(errMessage, "Too Many Requests"):
					common.SysLog(fmt.Sprintf("Too Many Requests %d to user %s: %v", activity.Id, u.ChatId, sendErr))
					queue.PushFront(u)
					time.Sleep(1 * time.Second)
					return
				case strings.Contains(errMessage, "Forbidden"):
					common.SysLog(fmt.Sprintf("Forbidden %d to user %s: %v", activity.Id, u.ChatId, sendErr))
					stats.IncrementFailed()
					model.UpdateUserStatusById(int(u.Id), 0)
					return
				default:
					common.SysLog(fmt.Sprintf("Error sending %d to user %s: %v", activity.Id, u.ChatId, sendErr))
					stats.IncrementFailed()
					return
				}
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
		"Push process %d:%s completed. Total users: %d, Success: %d, Failed: %d",
		activity.Id, activity.ShopId, stats.TotalUsers, stats.SuccessfulPush, stats.FailedPush,
	))
	common.SysLog(fmt.Sprintf(
		"Push process %d: startTime: %s, endTime: %s",
		activity.Id, stats.PushStartTime, stats.PushEndTime,
	))
}

// =================== PreviewMessage (HTTP API) ===================

func PreviewMessage(c *gin.Context) {
	uid := c.Param("uid")
	id := c.Param("id")

	activeID, err := strconv.Atoi(id)
	if err != nil {
		c.JSON(400, gin.H{"success": false, "message": "Invalid ID format"})
		return
	}

	activity, err := model.GetActiveContentByID(activeID, false)
	if err != nil {
		c.JSON(500, gin.H{"success": false, "message": "Error getting active content"})
		return
	}

	buttons, err := model.GetButtonsByActivityId(activeID)
	if err != nil {
		c.JSON(500, gin.H{"success": false, "message": "Error getting buttons"})
		return
	}

	userID, err := strconv.Atoi(uid)
	if err != nil {
		c.JSON(400, gin.H{"success": false, "message": "Invalid UID format"})
		return
	}

	user, err := model.GetUserById(userID, false)
	if err != nil {
		c.JSON(500, gin.H{"success": false, "message": "Error getting user"})
		return
	}

	bot, err := tgbotapi.NewBotAPI(os.Getenv("TELEGRAM_BOT_TOKEN"))
	if err != nil {
		c.JSON(500, gin.H{"success": false, "message": "Error creating bot"})
		return
	}

	user.Name = "预览用户"
	err = sendTelegramMessage(bot, user, activity, buttons)
	if err != nil {
		c.JSON(500, gin.H{"success": false, "message": err.Error()})
		return
	}

	c.JSON(200, gin.H{"success": true, "message": "Message sent successfully"})
}

// =================== Telegram 发送消息 ===================

func sendTelegramMessage(bot *tgbotapi.BotAPI, u *model.User, activity *model.Activity, buttons []*model.Button) error {
	chatID, err := strconv.ParseInt(u.ChatId, 10, 64)
	if err != nil {
		common.SysError(fmt.Sprintf("Error parsing chat ID for user %d: %v", u.Id, err))
		return err
	}

	var images []string
	err = json.Unmarshal([]byte(activity.Image), &images)
	if err != nil {
		return err
	}

	if len(images) > 0 && activity.Type == 0 {
		photo := tgbotapi.NewPhoto(chatID, tgbotapi.FileURL(os.Getenv("APP_IMAGE_BASE_URL")+"/uploads/"+images[0]))
		photo.Caption = "亲爱的" + common.FilterName(u.Name) + ":\n\n" + common.Text(activity.Content)
		photo.ParseMode = "HTML"
		if len(buttons) > 0 {
			photo.ReplyMarkup = tgbotapi.InlineKeyboardMarkup{InlineKeyboard: buildButtonOptions(buttons)}
		}
		sentMsgRes, err := bot.Send(photo)
		if err != nil {
			return err
		}
		if activity.IsPin == 1 {
			pinConfig := tgbotapi.PinChatMessageConfig{ChatID: chatID, MessageID: sentMsgRes.MessageID, DisableNotification: false}
			_, _ = bot.Request(pinConfig)
			time.Sleep(500 * time.Millisecond)
		}
	} else if activity.Type == 1 {
		video := tgbotapi.NewVideo(chatID, tgbotapi.FileURL(os.Getenv("APP_IMAGE_BASE_URL")+"/uploads/"+activity.Video))
		video.Caption = "亲爱的" + common.FilterName(u.Name) + ":\n\n" + common.Text(activity.Content)
		video.ParseMode = "HTML"
		if len(buttons) > 0 {
			video.ReplyMarkup = tgbotapi.InlineKeyboardMarkup{InlineKeyboard: buildButtonOptions(buttons)}
		}
		sentMsgRes, err := bot.Send(video)
		if err != nil {
			return err
		}
		if activity.IsPin == 1 {
			pinConfig := tgbotapi.PinChatMessageConfig{ChatID: chatID, MessageID: sentMsgRes.MessageID, DisableNotification: false}
			_, _ = bot.Request(pinConfig)
			time.Sleep(500 * time.Millisecond)
		}
	}

	return nil
}

// =================== 构建按钮 ===================

func buildButtonOptions(buttons []*model.Button) [][]tgbotapi.InlineKeyboardButton {
	maxLine := 0
	for _, button := range buttons {
		if button.OneLine > maxLine {
			maxLine = button.OneLine
		}
	}

	var options [][]tgbotapi.InlineKeyboardButton
	for line := 1; line <= maxLine; line++ {
		var lineOption []tgbotapi.InlineKeyboardButton
		for _, button := range buttons {
			if button.OneLine == line {
				lineOption = append(lineOption, buildButton(button))
			}
		}
		if len(lineOption) > 0 {
			options = append(options, lineOption)
		}
	}

	return options
}

func buildButton(button *model.Button) tgbotapi.InlineKeyboardButton {
	if button.Inline != "" {
		return tgbotapi.NewInlineKeyboardButtonData(button.Text, button.Inline)
	}
	buttonLink := button.Link
	if buttonLink == "" {
		buttonLink = "https://t.me/Ytxzs_bot"
	}
	return tgbotapi.NewInlineKeyboardButtonURL(button.Text, buttonLink)
}

// =================== 推送时间计算 ===================

func calculatePushJobStopDuration(currentActivity *model.Activity) time.Duration {
	pushDuration := common.PushJobStopDuration

	if currentActivity != nil && currentActivity.CountTime > 0 {
		pushDuration = time.Duration(currentActivity.CountTime) * time.Minute
		common.SysLog(fmt.Sprintf("Using count_time from current activity: %d minutes", currentActivity.CountTime))
		return pushDuration
	}

	rows, err := model.GetAllActivitiesOrderByTime()
	if err != nil {
		return pushDuration
	}

	var times []time.Time
	layout := "15:04:05"

	for _, activity := range rows {
		if activity.ActivityTime == "" {
			continue
		}
		parsedTime, err := time.Parse(layout, activity.ActivityTime)
		if err != nil {
			continue
		}
		times = append(times, parsedTime.UTC())
	}

	if len(times) == 0 {
		return pushDuration
	}

	sort.Slice(times, func(i, j int) bool {
		return times[i].Before(times[j])
	})

	currentTime := time.Now()
	currentTime = time.Date(0, 1, 1, currentTime.Hour(), currentTime.Minute(), currentTime.Second(), 0, time.UTC)

	for _, t := range times {
		if currentTime.Before(t) {
			pushDuration = t.Sub(currentTime)
			common.SysLog(fmt.Sprintf("Next push time: %s, duration: %d seconds", t.String(), pushDuration.Seconds()))
			return pushDuration
		}
	}

	nextDayFirstTime := times[0].Add(24 * time.Hour)
	pushDuration = nextDayFirstTime.Sub(currentTime)
	common.SysLog(fmt.Sprintf("Next push time: %s, duration: %d seconds", nextDayFirstTime.String(), pushDuration.Seconds()))
	return pushDuration
}