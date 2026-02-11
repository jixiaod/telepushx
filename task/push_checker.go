package task

import (
	"fmt"
	"sync/atomic"
	"time"

	"telepushx/common"
	"telepushx/controller"
	"telepushx/model"
)

// 批次锁，保证同一时间点只执行一个批次
var batchLock int32 = 0

// 检查数据库并推送消息
func CheckDatabaseAndPush() {
	// 如果已有批次在执行，跳过
	if !atomic.CompareAndSwapInt32(&batchLock, 0, 1) {
		common.SysLog("Previous batch still running, skip this check")
		return
	}
	defer atomic.StoreInt32(&batchLock, 0)

	now := time.Now()
	currentTime := now.Format("15:04:00")

	// 先将过期活动置 status=0
	if err := model.ExpireActivitiesByTime(now, currentTime); err != nil {
		common.SysError(fmt.Sprintf("Expire activities error: %v", err))
	}

	// 获取当前时间点有效活动
	activities, err := model.GetActivitiesByActivityTimeValid(currentTime, now)
	if err != nil {
		common.SysError(fmt.Sprintf("Query activities error: %v", err))
		return
	}
	if len(activities) == 0 {
		return
	}

	// 构建 region -> activityId 映射
	regionActivities := make(map[int][]int)
	descCache := make(map[int][]int)
	globalTargets := []int{}
	globalLoaded := false

	for _, a := range activities {
		targets := []int{}
		switch a.TargetScope {
		case 2:
			if !globalLoaded {
				ids, e := model.GetAllUserRegionIds()
				if e != nil {
					common.SysError(fmt.Sprintf("Error querying global regions: %v", e))
					continue
				}
				globalTargets = ids
				globalLoaded = true
			}
			targets = globalTargets

		case 1:
			rid := a.RegionId
			if rid == 0 {
				if !globalLoaded {
					ids, e := model.GetAllUserRegionIds()
					if e != nil {
						common.SysError(fmt.Sprintf("Error querying global regions: %v", e))
						continue
					}
					globalTargets = ids
					globalLoaded = true
				}
				targets = globalTargets
			} else {
				if cached, ok := descCache[rid]; ok {
					targets = cached
				} else {
					ids, e := model.GetDescendantRegionIds(rid)
					if e != nil {
						common.SysError(fmt.Sprintf("Error querying descendants: %v", e))
						continue
					}
					descCache[rid] = ids
					targets = ids
				}
			}

		default:
			if a.RegionId != 0 {
				targets = []int{a.RegionId}
			}
		}

		for _, tr := range targets {
			if tr == 0 {
				continue
			}
			regionActivities[tr] = append(regionActivities[tr], a.Id)
		}
	}

	// 每个目标地区推送轮询选择的活动
	for regionId, activityIds := range regionActivities {
		if len(activityIds) == 0 {
			continue
		}
		selectedActivityId := dailyRoundRobin(activityIds)
		common.SysLog(fmt.Sprintf("Region %d select activity %d at %s", regionId, selectedActivityId, currentTime))
		go controller.PushMessageByJob(selectedActivityId, regionId)
	}
}

// 启动定时任务，每分钟检测一次
func StartPushChecker() {
	nextMinute := time.Now().Truncate(time.Minute).Add(time.Minute)
	time.Sleep(time.Until(nextMinute))

	ticker := time.NewTicker(1 * time.Minute)
	common.SysLog("Push checker started")

	for range ticker.C {
		CheckDatabaseAndPush()
	}
}

// 每日轮询选择活动
func dailyRoundRobin(elements []int) int {
	today := time.Now().Unix() / (60 * 60 * 24)
	return elements[int(today)%len(elements)]
}