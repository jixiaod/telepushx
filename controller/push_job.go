package controller

import (
	"context"
	"fmt"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"telepushx/common"
	"telepushx/model"

	"golang.org/x/time/rate"
)

const (
	pushJobWorkerCount = 50
	pushRetryDelay     = 2 * time.Second
	pushTimeoutPadding = 30 * time.Second
	minPushJobTimeout  = 5 * time.Second
)

type pushJobRunner struct {
	activity       *model.Activity
	buttons        []*model.Button
	targetRegionID int
	stats          *common.PushStats
	queue          *model.UserQueue
	limiter        *rate.Limiter
}

// PushMessageByJob 根据活动id和目标地区推送消息
func PushMessageByJob(id int, targetRegionID int) {
	activity, buttons, err := loadPushActivity(id)
	if err != nil {
		common.SysError(err.Error())
		return
	}

	go newPushJobRunner(activity, buttons, targetRegionID).run()
}

func loadPushActivity(id int) (*model.Activity, []*model.Button, error) {
	activity, err := model.GetActiveContentByID(id, false)
	if err != nil {
		return nil, nil, fmt.Errorf("error getting activity %d: %w", id, err)
	}

	buttons, err := model.GetButtonsByActivityId(id)
	if err != nil {
		return nil, nil, fmt.Errorf("error getting buttons for activity %d: %w", id, err)
	}

	return activity, buttons, nil
}

func newPushJobRunner(activity *model.Activity, buttons []*model.Button, targetRegionID int) *pushJobRunner {
	return &pushJobRunner{
		activity:       activity,
		buttons:        buttons,
		targetRegionID: targetRegionID,
		limiter:        buildPushLimiter(activity),
	}
}

func (r *pushJobRunner) run() {
	users, err := model.GetAllUsersWithRegionId(r.targetRegionID, 0, common.GetAllUsersLimitSizeNum)
	if err != nil {
		common.SysError(fmt.Sprintf("Error getting users: %v", err))
		return
	}

	if len(users) == 0 {
		common.SysLog(fmt.Sprintf("No users in region %d for activity %d", r.targetRegionID, r.activity.Id))
		return
	}

	r.stats = common.NewPushStats(len(users))
	r.stats.RecordStartTime()
	r.queue = &model.UserQueue{}
	r.queue.PushBatch(users)

	common.SysLog(fmt.Sprintf(
		"Start pushing activity %d to %d users (activityRegion: %d, targetRegion: %d)",
		r.activity.Id, len(users), r.activity.RegionId, r.targetRegionID,
	))

	bot, err := newTelegramBot()
	if err != nil {
		common.SysError(fmt.Sprintf("Error creating bot: %v", err))
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), calculatePushJobTimeout(r.activity))
	defer cancel()

	r.dispatch(ctx, bot)
	r.stats.RecordEndTime()

	common.SysLog(fmt.Sprintf(
		"Push completed activity %d (region %d): Total=%d, Success=%d, Failed=%d",
		r.activity.Id, r.targetRegionID, r.stats.TotalUsers, r.stats.SuccessfulPush, r.stats.FailedPush,
	))
}

func (r *pushJobRunner) dispatch(ctx context.Context, bot telegramBot) {
	var wg sync.WaitGroup
	sem := make(chan struct{}, pushJobWorkerCount)

dispatchLoop:
	for {
		user := r.queue.Pop()
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
			defer recoverPushPanic(r.activity.Id, u.ChatId)

			r.processUser(ctx, bot, u)
		}(user)

		if !r.queue.HasNext() {
			break
		}
	}

	wg.Wait()
}

func (r *pushJobRunner) processUser(ctx context.Context, bot telegramBot, user *model.User) {
	if ctx.Err() != nil {
		return
	}

	if err := r.limiter.Wait(ctx); err != nil {
		r.queue.PushFront(user)
		return
	}

	sendErr := sendTelegramMessage(bot, user, r.activity, r.buttons)
	if sendErr == nil {
		r.stats.IncrementSuccess()
		return
	}

	r.handleSendError(user, sendErr)
}

func (r *pushJobRunner) handleSendError(user *model.User, err error) {
	errMsg := err.Error()

	switch {
	case strings.Contains(errMsg, "Gateway Timeout"), strings.Contains(errMsg, "Too Many Requests"):
		r.queue.PushFront(user)
		time.Sleep(pushRetryDelay)
	case strings.Contains(errMsg, "Forbidden"):
		r.stats.IncrementFailed()
		_ = model.UpdateUserStatusById(int(user.Id), 0)
	default:
		r.stats.IncrementFailed()
	}
}

func buildPushLimiter(activity *model.Activity) *rate.Limiter {
	limit := common.PushJobRateLimitNum
	if activity.IsPin == 1 {
		limit = common.PinPushJobRateLimitNum
	}

	return rate.NewLimiter(rate.Limit(limit), 1)
}

func calculatePushJobTimeout(activity *model.Activity) time.Duration {
	timeout := calculatePushJobStopDuration(activity) - pushTimeoutPadding
	if timeout < minPushJobTimeout {
		return minPushJobTimeout
	}

	return timeout
}

func recoverPushPanic(activityID int, chatID string) {
	if recovered := recover(); recovered != nil {
		common.SysError(fmt.Sprintf(
			"panic in push goroutine activity=%d user=%s r=%v",
			activityID, chatID, recovered,
		))
		common.SysError(string(debug.Stack()))
	}
}
