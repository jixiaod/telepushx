package common

import "time"

var StartTime = time.Now().Unix() // unit: second
var Version = "v0.0.1"
var SystemName = "消息推送服务"
var ServerAddress = "http://localhost:3000"

var TelegramAPIBaseURL = "https://api.telegram.org/bot"
var TelegramBotToken = "6253545273:AAEzdFppjluWM_QplxoZiMsi0GJzDgNmEVI"
var TelegramBotName = "消息推送服务"
var TelegramBotAvatar = ""
var TelegramBotDescription = "消息推送服务"
var TelegramBotHomePageLink = "https://tiger.ytxzs.com"
var TelegramBotFooter = ""
var TelegramBotFooterLink = "https://tiger.ytxzs.com"

// All duration's unit is seconds
// Shouldn't larger then RateLimitKeyExpirationDuration
var (
	GlobalApiRateLimitNum            = 60
	GlobalApiRateLimitDuration int64 = 3 * 60

	GlobalWebRateLimitNum            = 60
	GlobalWebRateLimitDuration int64 = 3 * 60

	UploadRateLimitNum            = 10
	UploadRateLimitDuration int64 = 60

	DownloadRateLimitNum            = 10
	DownloadRateLimitDuration int64 = 60

	CriticalRateLimitNum            = 20
	CriticalRateLimitDuration int64 = 20 * 60
)

var RateLimitKeyExpirationDuration = 20 * time.Minute