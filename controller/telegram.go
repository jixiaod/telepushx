package controller

import (
	"encoding/json"
	"os"
	"sort"
	"strconv"
	"time"

	"telepushx/common"
	"telepushx/model"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const defaultButtonURL = "https://t.me/Ytxzs_bot"

type telegramBot interface {
	Send(c tgbotapi.Chattable) (tgbotapi.Message, error)
	Request(c tgbotapi.Chattable) (*tgbotapi.APIResponse, error)
}

func newTelegramBot() (*tgbotapi.BotAPI, error) {
	return tgbotapi.NewBotAPI(os.Getenv("TELEGRAM_BOT_TOKEN"))
}

func sendTelegramMessage(bot telegramBot, user *model.User, activity *model.Activity, buttons []*model.Button) error {
	chatID, err := strconv.ParseInt(user.ChatId, 10, 64)
	if err != nil {
		return err
	}

	messageText := buildMessageCaption(user, activity)
	replyMarkup := buildReplyMarkup(buttons)

	switch activity.Type {
	case 1:
		return sendVideoMessage(bot, chatID, activity, messageText, replyMarkup)
	default:
		return sendPhotoMessage(bot, chatID, activity, messageText, replyMarkup)
	}
}

func buildMessageCaption(user *model.User, activity *model.Activity) string {
	return "亲爱的" + common.FilterName(user.Name) + ":\n\n" + common.Text(activity.Content)
}

func sendPhotoMessage(bot telegramBot, chatID int64, activity *model.Activity, caption string, replyMarkup *tgbotapi.InlineKeyboardMarkup) error {
	images, err := parseActivityImages(activity.Image)
	if err != nil {
		return err
	}
	if len(images) == 0 {
		return nil
	}

	photo := tgbotapi.NewPhoto(chatID, tgbotapi.FileURL(uploadFileURL(images[0])))
	photo.Caption = caption
	photo.ParseMode = "HTML"
	if replyMarkup != nil {
		photo.ReplyMarkup = *replyMarkup
	}

	sentMsg, err := bot.Send(photo)
	if err != nil {
		return err
	}

	return pinMessageIfNeeded(bot, activity, chatID, sentMsg.MessageID)
}

func sendVideoMessage(bot telegramBot, chatID int64, activity *model.Activity, caption string, replyMarkup *tgbotapi.InlineKeyboardMarkup) error {
	video := tgbotapi.NewVideo(chatID, tgbotapi.FileURL(uploadFileURL(activity.Video)))
	video.Caption = caption
	video.ParseMode = "HTML"
	if replyMarkup != nil {
		video.ReplyMarkup = *replyMarkup
	}

	sentMsg, err := bot.Send(video)
	if err != nil {
		return err
	}

	return pinMessageIfNeeded(bot, activity, chatID, sentMsg.MessageID)
}

func parseActivityImages(raw string) ([]string, error) {
	var images []string
	if raw == "" {
		return images, nil
	}

	err := json.Unmarshal([]byte(raw), &images)
	return images, err
}

func uploadFileURL(fileName string) string {
	return os.Getenv("APP_IMAGE_BASE_URL") + "/uploads/" + fileName
}

func pinMessageIfNeeded(bot telegramBot, activity *model.Activity, chatID int64, messageID int) error {
	if activity.IsPin != 1 {
		return nil
	}

	_, _ = bot.Request(tgbotapi.PinChatMessageConfig{
		ChatID:    chatID,
		MessageID: messageID,
	})
	time.Sleep(500 * time.Millisecond)

	return nil
}

func buildReplyMarkup(buttons []*model.Button) *tgbotapi.InlineKeyboardMarkup {
	keyboard := buildButtonOptions(buttons)
	if len(keyboard) == 0 {
		return nil
	}

	return &tgbotapi.InlineKeyboardMarkup{InlineKeyboard: keyboard}
}

func buildButtonOptions(buttons []*model.Button) [][]tgbotapi.InlineKeyboardButton {
	rowsByLine := make(map[int][]tgbotapi.InlineKeyboardButton)
	lines := make([]int, 0)

	for _, button := range buttons {
		line := button.OneLine
		if _, exists := rowsByLine[line]; !exists {
			lines = append(lines, line)
		}
		rowsByLine[line] = append(rowsByLine[line], buildButton(button))
	}

	sort.Ints(lines)

	rows := make([][]tgbotapi.InlineKeyboardButton, 0, len(lines))
	for _, line := range lines {
		rows = append(rows, rowsByLine[line])
	}

	return rows
}

func buildButton(button *model.Button) tgbotapi.InlineKeyboardButton {
	if button.Inline != "" {
		return tgbotapi.NewInlineKeyboardButtonData(button.Text, button.Inline)
	}

	link := button.Link
	if link == "" {
		link = defaultButtonURL
	}

	return tgbotapi.NewInlineKeyboardButtonURL(button.Text, link)
}

func calculatePushJobStopDuration(activity *model.Activity) time.Duration {
	if activity != nil && activity.CountTime > 0 {
		return time.Duration(activity.CountTime) * time.Minute
	}

	rows, err := model.GetAllActivitiesOrderByTime()
	if err != nil || len(rows) == 0 {
		return common.PushJobStopDuration
	}

	times := make([]time.Time, 0, len(rows))
	for _, activityRow := range rows {
		parsedTime, ok := parseActivityClock(activityRow.ActivityTime)
		if ok {
			times = append(times, parsedTime)
		}
	}

	if len(times) == 0 {
		return common.PushJobStopDuration
	}

	sort.Slice(times, func(i, j int) bool { return times[i].Before(times[j]) })

	currentTime := currentClockTime()
	for _, candidate := range times {
		if currentTime.Before(candidate) {
			return candidate.Sub(currentTime)
		}
	}

	return times[0].Add(24 * time.Hour).Sub(currentTime)
}

func parseActivityClock(raw string) (time.Time, bool) {
	if raw == "" {
		return time.Time{}, false
	}

	parsed, err := time.Parse("15:04:05", raw)
	if err != nil {
		return time.Time{}, false
	}

	return parsed.UTC(), true
}

func currentClockTime() time.Time {
	now := time.Now()
	return time.Date(0, 1, 1, now.Hour(), now.Minute(), now.Second(), 0, time.UTC)
}
