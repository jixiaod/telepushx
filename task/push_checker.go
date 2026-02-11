package task

import (
	"fmt"
	"sync/atomic"
	"time"

	"telepushx/common"
	"telepushx/controller"
	"telepushx/model"
)

// 定时检查数据库并推送消息
func CheckDatabaseAndPush() {
	// 抢占推送锁，如果已有推送，跳过本轮
	if !atomic.CompareAndSwapInt32(&controller.PushMessageLock, 0, 1) {
		return
	}
	defer atomic.StoreInt32(&controller.PushMessageLock, 0)

	now := time.Now()
	currentTime := now.Format("15:04:00") // HH:mm:ss 格式

	// 将过期活动置 status=0
	if err := model.ExpireActivitiesByTime(now, currentTime); err != nil {
		common.SysError(fmt.Sprintf("Expire activities error: %v", err))
	}

	// 获取当前时间点的有效活动
	activities, err := model.GetActivitiesByActivityTimeValid(currentTime, now)
	if err != nil {
		common.SysError(fmt.Sprintf("Error querying activities: %v", err))
		return
	}

	if len(activities) == 0 {
		return
	}

	// 按 targetRegionId 分组
	regionActivities := make(map[int][]int)

	// 缓存 descendants 避免重复查询
	descCache := make(map[int][]int)

	// 全局目标地区缓存
	globalTargets := []int{}
	globalLoaded := false

	for _, a := range activities {
		targets := []int{}

		switch a.TargetScope {
		case 2: // 全局：推到所有“有用户的地区”
			if !globalLoaded {
				ids, e := model.GetAllUserRegionIds()
				if e != nil {
					common.SysError(fmt.Sprintf("Error querying global region ids: %v", e))
					continue
				}
				globalTargets = ids
				globalLoaded = true
			}
			targets = globalTargets

		case 1: // 含下级：把活动 region 展开为 descendants(region)
			rid := a.RegionId
			if rid == 0 {
				if !globalLoaded {
					ids, e := model.GetAllUserRegionIds()
					if e != nil {
						common.SysError(fmt.Sprintf("Error querying global region ids: %v", e))
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
						common.SysError(fmt.Sprintf("Error querying descendant regions: %v", e))
						continue
					}
					descCache[rid] = ids
					targets = ids
				}
			}

		default: // 仅本地区
			if a.RegionId != 0 {
				targets = []int{a.RegionId}
			}
		}

		// 分配 activityId 到每个 targetRegionId
		for _, tr := range targets {
			if tr == 0 {
				continue
			}
			regionActivities[tr] = append(regionActivities[tr], a.Id)
		}
	}

	// 每个目标地区单独轮询推送
	for targetRegionId, activityIds := range regionActivities {
		if len(activityIds) == 0 {
			continue
		}

		selectedActivityId := dailyRoundRobin(activityIds)

		common.SysLog(fmt.Sprintf(
			"Region %d select activity %d at %s",
			targetRegionId, selectedActivityId, currentTime,
		))

		go controller.PushMessageByJob(selectedActivityId, targetRegionId)
	}
}

// 启动定时任务，每分钟检查一次
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

// 日常轮询选择活动，按天索引循环
func dailyRoundRobin(elements []int) int {
	today := time.Now()
	dayIndex := today.Unix() / (60 * 60 * 24)
	currentIndex := int(dayIndex) % len(elements)
	return elements[currentIndex]
}