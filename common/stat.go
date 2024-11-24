package common

import (
	"sync/atomic"
	"time"
)

type PushStats struct {
	TotalUsers     int32
	SuccessfulPush int32
	FailedPush     int32
	PushStartTime  time.Time
	PushEndTime    time.Time
}

func (ps *PushStats) RecordStartTime() {
	ps.PushStartTime = time.Now()
}

func (ps *PushStats) RecordEndTime() {
	ps.PushEndTime = time.Now()
}

func NewPushStats(totalUsers int) *PushStats {
	return &PushStats{
		TotalUsers: int32(totalUsers),
	}
}

func (ps *PushStats) IncrementSuccess() {
	atomic.AddInt32(&ps.SuccessfulPush, 1)
}

func (ps *PushStats) IncrementFailed() {
	atomic.AddInt32(&ps.FailedPush, 1)
}

func (ps *PushStats) GetSuccessRate() float64 {
	totalUsers := atomic.LoadInt32(&ps.TotalUsers)
	if totalUsers == 0 {
		return 0
	}
	successfulPush := atomic.LoadInt32(&ps.SuccessfulPush)
	return float64(successfulPush) / float64(totalUsers) * 100
}
