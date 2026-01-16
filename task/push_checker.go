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
	currentTime := now.Format("15:04:00")

	// 先把过期的活动置 status=0
	if err := model.ExpireActivitiesByTime(now, currentTime); err != nil {
		common.SysError(fmt.Sprintf("Expire activities error: %v", err))
	}

	// 1) 查该时刻所有有效活动（不按 region 过滤）
    activities, err := model.GetActivitiesByActivityTimeValid(currentTime, now)
    if err != nil {
        common.SysError(fmt.Sprintf("Error querying activities: %v", err))
        return
    }

	// No activities to push
	if len(activities) == 0 {
		return
	}
	//  2) 按 region_id 分组
	// 展开：targetRegionId => []activityId
	regionActivities := make(map[int][]int)


    // descendants 缓存，避免重复查 closure
    descCache := make(map[int][]int)

    // 全局目标地区缓存（按 users distinct region_id）
    globalTargets := []int{}
    globalLoaded := false

    for _, a := range activities {

        targets := []int{}

        switch a.TargetScope {
        case 2:
            // 全局：推到所有“有用户的地区”
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

        case 1:
            // 含下级：把活动 region 展开为 descendants(region)
            rid := a.RegionId
            if rid == 0 {
                // 兜底：region_id=0 + scope=1，按全局处理
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

        default:
            // 仅本地区：只推到该 region 本身
            if a.RegionId != 0 {
                targets = []int{a.RegionId}
            }
        }

        // 3) 把 activityId 分配到每个 targetRegionId
        for _, tr := range targets {
            if tr == 0 {
                continue
            }
            regionActivities[tr] = append(regionActivities[tr], a.Id)
        }
    }



    // 4) 每个目标地区单独轮询推送
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
