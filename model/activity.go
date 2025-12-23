package model

import (
	"errors"
)

type Activity struct {
	Id           int    `gorm:"column:id;primaryKey;autoIncrement;type:int(10) unsigned"`
	RegionId     int    `gorm:"column:region_id;type:int(10) unsigned"`
	Content      string `gorm:"column:activity_text;type:text"`
	Image        string `gorm:"column:activity_image;type:text"`
	Video        string `gorm:"column:mp4;type:text"`
	Type         int    `gorm:"column:type;type:int(11) unsigned"`
	IsPin        int    `gorm:"column:is_pin;type:int(11) unsigned"`
	ShopId       string `gorm:"column:shop_id;type:varchar(255)"`
	ActivityTime string `gorm:"column:activity_time;type:varchar(60)"`
	CountTime    int    `gorm:"column:count_time;type:int(11) unsigned"`
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
