package model

import (
	"errors"
)

type Button struct {
	Id         int    `gorm:"column:id;primaryKey;autoIncrement;type:int(10) unsigned"`
	ActivityId int    `gorm:"column:activity_id;type:int(10) unsigned"`
	Text       string `gorm:"column:button_text;type:varchar(60)"`
	Link       string `gorm:"column:button_link;type:varchar(200)"`
	Inline     string `gorm:"column:button_inline;type:varchar(60)"`
	OneLine    int    `gorm:"column:one_line;type:int(10) unsigned"`
}

func (Button) TableName() string {
	return "activity_button"
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
		err = DB.Select([]string{"id", "activity_text as content", "activity_image as image"}).First(&activity, "id = ?", id).Error
	}
	return &activity, err
}
