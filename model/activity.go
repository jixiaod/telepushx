package model

import (
	"errors"
 	"time"
	"database/sql"
)

type Activity struct {
	Id           int    `gorm:"column:id;primaryKey;autoIncrement;type:int(10) unsigned"`
	RegionId     int    `gorm:"column:region_id;type:int(10) unsigned"`
	TargetScope  int    `gorm:"column:target_scope"` //
	Content      string `gorm:"column:activity_text;type:text"`
	Image        string `gorm:"column:activity_image;type:text"`
	Video        string `gorm:"column:mp4;type:text"`
	Type         int    `gorm:"column:type;type:int(11) unsigned"`
	IsPin        int    `gorm:"column:is_pin;type:int(11) unsigned"`
	ShopId       string `gorm:"column:shop_id;type:varchar(255)"`
	ActivityTime string `gorm:"column:activity_time;type:varchar(60)"`
	CountTime    int    `gorm:"column:count_time;type:int(11) unsigned"`
	StartDate    sql.NullTime `gorm:"column:start_date;type:date"`
    EndDate      sql.NullTime `gorm:"column:end_date;type:date"`
}

func (Activity) TableName() string {
	return "activity"
}

func GetActiveContentByID(id int, selectAll bool) (*Activity, error) {
	if id == 0 {
		return nil, errors.New("id 为空！")
	}
	activity := Activity{Id: id}
	var err error = nil
	if selectAll {
		err = DB.First(&activity, "id = ?", id).Error
	} else {
		err = DB.Select([]string{"id", "region_id", "activity_text", "activity_image", "mp4", "type", "is_pin", "shop_id", "count_time"}).First(&activity, "id = ?", id).Error
	}
	return &activity, err
}

func GetAllActivitiesOrderByTime() ([]*Activity, error) {
	var activities []*Activity
	err := DB.Where("status = 1").Order("activity_time ASC").Find(&activities).Error
	return activities, err
}

func GetActivitiesByActivityTime(currentTime string) ([]*Activity, error) {
	var activities []*Activity
	//err := DB.Where("status = 1").Where("activity_time = ?", currentTime).Find(&activities).Error
	err := DB.Where("status = 1").Where("DATE_FORMAT(activity_time, '%H:%i:00') = ?", currentTime).Find(&activities).Error

	return activities, err
}

func GetAllActivities() ([]*Activity, error) {
	var activities []*Activity
	err := DB.Find(&activities).Error
	return activities, err
}

func GetActivitiesByActivityTimeValid(currentTime string, today time.Time) ([]Activity, error) {
    // todayStr: "YYYY-MM-DD"
    todayStr := today.Format("2006-01-02")

    var activities []Activity
    err := DB.
		//Debug().
        Where("status = 1").
        Where("activity_time = ?", currentTime).
        // 有效期过滤（DATE）
        Where("start_date IS NULL OR start_date <= ?", todayStr).
        Where("end_date IS NULL OR end_date >= ?", todayStr).
        Find(&activities).Error

    return activities, err
}

func ExpireActivitiesByTime(today time.Time, currentTime string) error {
    todayStr := today.Format("2006-01-02")

    // 将 end_date < today 的活动置为 status=0
    return DB.Model(&Activity{}).
        Where("status = 1").
        Where("activity_time = ?", currentTime).
        Where("end_date IS NOT NULL AND end_date < ?", todayStr).
        Update("status", 0).Error
}
