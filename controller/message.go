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
	"telepushx/common"
	"telepushx/model"
	"time"

	"github.com/gin-gonic/gin"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"golang.org/x/time/rate"
)

// 全局变量，标识是否有正在进行中的推送
var IsPushingMessage bool = false

func PushMessage(c *gin.Context) {
	if IsPushingMessage {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Push service is running now, please wait.",
			"data":    gin.H{},
		})
	}

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

	buttons, err := model.GetButtonsByActivityId(intId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Error getting active buttons",
			"data":    gin.H{},
		})
		return
	}
	go doPushMessage(activity, buttons)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Push process started",
		"data":    gin.H{},
	})
	//return nil
}

func PushMessageByJob(id int) {

	activity, err := model.GetActiveContentByID(id, false)
	if err != nil {

		return
	}

	buttons, err := model.GetButtonsByActivityId(id)
	if err != nil {

		return
	}
	go doPushMessage(activity, buttons)
}

func doPushMessage(activity *model.Activity, buttons []*model.Button) {

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
	stats.RecordStartTime()
	limiter := rate.NewLimiter(rate.Limit(common.PushJobRateLimitNum), 1)
	if activity.IsPin == 1 {
		limiter = rate.NewLimiter(rate.Limit(common.PinPushJobRateLimitNum), 1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), calculatePushJobStopDuration()-30*time.Second)
	defer cancel()

	var wg sync.WaitGroup

	queue := &model.UserQueue{}
	queue.PushBatch(users)

	IsPushingMessage = true

	// 遍历队列中的用户
	queue.ForEach(func(user *model.User) {

		wg.Add(1)
		go func(u *model.User) {

			defer wg.Done()

			select {
			case <-ctx.Done():
				return
			default:
				if err := limiter.Wait(ctx); err != nil {
					// If rate limit is exceeded, add the user back to the front of the queue
					//users = append([]*model.User{u}, users...)
					queue.PushFront(u)
					//common.SysLog(fmt.Sprintf("Rate limit exceeded for user %s adding back to the front of the queue", u.ChatId))
					return
				}

				err = sendTelegramMessage(bot, u, activity, buttons)

				if err != nil {
					errMessage := err.Error() // 缓存错误消息，避免重复调用
					if strings.Contains(errMessage, "Too Many Requests") {
						common.SysLog(fmt.Sprintf("Too Many Requests %d to user %s: %v", activity.Id, u.ChatId, err))
						queue.PushFront(u) // Re-add user to the front of the queue
					} else {
						common.SysLog(fmt.Sprintf("Error sending %d to user %s: %v", activity.Id, u.ChatId, err))
						stats.IncrementFailed()

						if strings.Contains(errMessage, "Forbidden") {
							model.UpdateUserStatusById(int(u.Id), 0)
						}
					}
				} else {
					//common.SysLog(fmt.Sprintf("Message sent successfully to user %s", u.ChatId))
					stats.IncrementSuccess()
				}
				return
			}
		}(user)

	})

	wg.Wait()
	stats.RecordEndTime()
	common.SysLog("Push process completed.")
	common.SysLog(fmt.Sprintf("Push process %d:%s completed. Total users: %d, Success: %d, Failed: %d", activity.Id, activity.ShopId, stats.TotalUsers, stats.SuccessfulPush, stats.FailedPush))
	common.SysLog(fmt.Sprintf("Push process %d: startTime: %s, endTime: %s", activity.Id, stats.PushStartTime, stats.PushEndTime))
	IsPushingMessage = false
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

func sendTelegramMessage(bot *tgbotapi.BotAPI, u *model.User, activity *model.Activity, buttons []*model.Button) (err error) {
	chatID, err := strconv.ParseInt(u.ChatId, 10, 64)
	if err != nil {
		common.SysError(fmt.Sprintf("Error parsing image JSON for user %d %s: %v", u.Id, u.ChatId, err))
		return err
	}
	var images []string
	err = json.Unmarshal([]byte(activity.Image), &images)
	if err != nil {
		//common.SysError(fmt.Sprintf("Error parsing image JSON for user %s: %v", u.ChatId, err))
		return err
	}

	if len(images) > 0 && activity.Type == 0 {
		photo := tgbotapi.NewPhoto(chatID, tgbotapi.FileURL(os.Getenv("APP_IMAGE_BASE_URL")+"/uploads/"+images[0]))
		photo.Caption = "亲爱的" + common.FilterName(u.Name) + ":\n\n" + common.Text(activity.Content)
		photo.ParseMode = "HTML"
		if len(buttons) > 0 {
			inlineKeyboard := buildButtonOptions(buttons)
			photo.ReplyMarkup = tgbotapi.InlineKeyboardMarkup{
				InlineKeyboard: inlineKeyboard,
			}
		}
		sentMsgRes, err := bot.Send(photo)
		if err != nil {
			//common.SysLog(fmt.Sprintf("Error sending photo message to user %s: %v", u.ChatId, err))
			return err
		}

		if activity.IsPin == 1 {
			// 置顶消息
			pinConfig := tgbotapi.PinChatMessageConfig{
				ChatID:              chatID,
				MessageID:           sentMsgRes.MessageID,
				DisableNotification: false, // true 表示静默置顶
			}
			_, err = bot.Request(pinConfig)
			if err != nil {
				common.SysLog(fmt.Sprintf("置顶消息失败: %v", err))
			}
			time.Sleep(500 * time.Millisecond)
		}

	} else if activity.Type == 1 {
		video := tgbotapi.NewVideo(chatID, tgbotapi.FileURL(os.Getenv("APP_IMAGE_BASE_URL")+"/uploads/"+activity.Video))
		video.Caption = "亲爱的" + common.FilterName(u.Name) + ":\n\n" + common.Text(activity.Content)
		video.ParseMode = "HTML"
		if len(buttons) > 0 {
			inlineKeyboard := buildButtonOptions(buttons)
			video.ReplyMarkup = tgbotapi.InlineKeyboardMarkup{
				InlineKeyboard: inlineKeyboard,
			}
		}
		sentMsgRes, err := bot.Send(video)
		if err != nil {
			//common.SysLog(fmt.Sprintf("Error sending video message to user %s: %v", u.ChatId, err))
			return err
		}

		if activity.IsPin == 1 {
			// 置顶消息
			pinConfig := tgbotapi.PinChatMessageConfig{
				ChatID:              chatID,
				MessageID:           sentMsgRes.MessageID,
				DisableNotification: false, // true 表示静默置顶
			}
			_, err = bot.Request(pinConfig)
			if err != nil {
				common.SysLog(fmt.Sprintf("置顶消息失败: %v", err))
			}
			time.Sleep(500 * time.Millisecond)
		}
	}

	return nil
}

func buildButtonOptions(buttons []*model.Button) [][]tgbotapi.InlineKeyboardButton {
	// 找出最大行数
	maxLine := 0
	for _, button := range buttons {
		if button.OneLine > maxLine {
			maxLine = button.OneLine
		}
	}

	var options [][]tgbotapi.InlineKeyboardButton
	if maxLine > 0 {
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
	}

	return options
}

func buildButton(button *model.Button) tgbotapi.InlineKeyboardButton {
	if button.Inline != "" {
		// 处理 inline callback 按钮
		return tgbotapi.NewInlineKeyboardButtonData(
			button.Text,
			button.Inline,
		)
	} else {
		// 处理 URL 按钮
		buttonLink := button.Link
		if buttonLink == "" {
			buttonLink = "https://t.me/Ytxzs_bot"
		}
		return tgbotapi.NewInlineKeyboardButtonURL(
			button.Text,
			buttonLink,
		)
	}
}

func CalculatePushTime(c *gin.Context) {

	duration := calculatePushJobStopDuration()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Calculated push duration",
		"data":    duration,
	})
}

func calculatePushJobStopDuration() time.Duration {

	pushDuration := common.PushJobStopDuration
	// 查询 activity_time 数据
	rows, err := model.GetAllActivitiesOrderByTime()
	if err != nil {
		return pushDuration
	}

	var times []time.Time
	layout := "15:04:05" // 时间格式
	// 读取并解析数据库中的时间
	for _, activity := range rows {
		timeStr := activity.ActivityTime

		if timeStr == "" {
			continue
		}
		parsedTime, err := time.Parse(layout, timeStr)
		parsedTime = parsedTime.UTC()
		if err != nil {
			continue
		}
		times = append(times, parsedTime)
	}

	// 检查是否有推送时间
	if len(times) == 0 {
		return pushDuration
	}

	// 按时间排序
	sort.Slice(times, func(i, j int) bool {
		return times[i].Before(times[j])
	})

	// 获取当前时间的时分秒部分
	currentTime := time.Now()
	currentTime = time.Date(0, 1, 1, currentTime.Hour(), currentTime.Minute(), currentTime.Second(), 0, time.UTC)

	common.SysLog(fmt.Sprintf("Current time: %s", currentTime.String()))

	// 找到当前时间之后的下一条推送时间
	for _, t := range times {
		if currentTime.Before(t) {

			pushDuration = t.Sub(currentTime)
			common.SysLog(fmt.Sprintf("Next push time: %s, duration: %d", t.String(), pushDuration/1000000000))
			return pushDuration
		}
	}

	// 如果没有找到下一条时间，则当前时间已经是最后一条推送，返回到次日第一条推送时间
	nextDayFirstTime := times[0].Add(24 * time.Hour)
	pushDuration = nextDayFirstTime.Sub(currentTime)
	common.SysLog(fmt.Sprintf("Next push time: %s, duration: %d", nextDayFirstTime, pushDuration/1000000000))
	return pushDuration
}
