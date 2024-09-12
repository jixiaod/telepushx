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

func GetButtonsByActivityId(activityId int) ([]*Button, error) {
	if activityId == 0 {
		return nil, errors.New("activity id 为空！")
	}
	var buttons []*Button
	err := DB.Where("activity_id = ?", activityId).Find(&buttons).Error
	return buttons, err
}
