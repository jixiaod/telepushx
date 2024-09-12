package model

import (
	"errors"
)

type Activity struct {
	Id      int    `gorm:"column:id;primaryKey;autoIncrement;type:int(10) unsigned"`
	Content string `gorm:"column:activity_text;type:text"`
	Image   string `gorm:"column:activity_image;type:text"`
	Video   string `gorm:"column:activity_video;type:text"`
	Type    int    `gorm:"column:type;type:int(11) unsigned"`
	ShopId  string `gorm:"column:shop_id;type:varchar(255)"`
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
		err = DB.Select([]string{"id", "activity_text", "activity_image", "type", "shop_id"}).First(&activity, "id = ?", id).Error
	}
	return &activity, err
}
