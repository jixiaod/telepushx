package common

import "time"

var StartTime = time.Now().Unix() // unit: second
var Version = "v0.0.1"
var SystemName = "Telegram Message Push Service"
var ServerAddress = "http://localhost:3000"

// All duration's unit is seconds
// Shouldn't larger then RateLimitKeyExpirationDuration
var (
	GlobalApiRateLimitNum            = 600
	GlobalApiRateLimitDuration int64 = 3 * 60

	GlobalWebRateLimitNum            = 600
	GlobalWebRateLimitDuration int64 = 3 * 60

	UploadRateLimitNum            = 10
	UploadRateLimitDuration int64 = 60

	DownloadRateLimitNum            = 10
	DownloadRateLimitDuration int64 = 60

	CriticalRateLimitNum            = 200
	CriticalRateLimitDuration int64 = 20 * 60
)

var RateLimitKeyExpirationDuration = 20 * time.Minute

var GetAllUsersLimitSizeNum = 100000
var PushJobStopDuration = 20 * time.Minute
var PushJobRateLimitNum = 25
var PinPushJobRateLimitNum = 10
