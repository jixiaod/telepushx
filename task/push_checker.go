package task

import (
	"fmt"

	"telepushx/common"
	"telepushx/controller"
	"telepushx/model"
	"time"
)

// 定义定时任务逻辑
func CheckDatabaseAndPush() {
	if controller.IsPushingMessage {
		return
	}

	//common.SysLog(fmt.Sprintf("Checking database for pending push tasks:%v", time.Now()))
	// Get current time and format to HH:mm
	now := time.Now()
	currentTime := now.Format("15:04")

	// Query activities scheduled for current time
	activities, err := model.GetActivitiesByActivityTime(currentTime + ":00")
	if err != nil {
		common.SysError(fmt.Sprintf("Error querying activities: %v", err))
		return
	}

	// No activities to push
	if len(activities) == 0 {
		return
	}
	selectedActivityIds := make([]int, len(activities))
	for i, activity := range activities {
		selectedActivityIds[i] = activity.Id
	}
	selectedActivityId := dailyRoundRobin(selectedActivityIds)

	go controller.PushMessageByJob(selectedActivityId)

}

// 定时任务启动函数
func StartPushChecker() {
	nextMinute := time.Now().Truncate(time.Minute).Add(time.Minute)
	waitDuration := time.Until(nextMinute)
	time.Sleep(waitDuration)

	doStartPushChecker()

}

func doStartPushChecker() {
	ticker := time.NewTicker(1 * time.Minute)
	common.SysLog(fmt.Sprintf("Push checker started at %v", time.Now()))
	go func() {
		for {
			select {
			case <-ticker.C:
				CheckDatabaseAndPush()
			}
		}
	}()
}

func dailyRoundRobin(elements []int) int {
	// 获取当前日期
	today := time.Now()
	// 将日期转换为天数
	dayIndex := today.Unix() / (60 * 60 * 24)
	// 计算当前索引
	currentIndex := int(dayIndex) % len(elements)

	return elements[currentIndex]
}
