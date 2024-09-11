package model

import (
	"errors"
)

type Activity struct {
	Id      int
	Content string
	Image   string
}

type Button struct {
	Text string
	Link string
}

func GetActiveContentByID(id int, selectAll bool) (*Activity, error) {
	if id == 0 {
		return nil, errors.New("id 为空！")
	}
	activity := Activity{Id: id}
	var err error = nil
	if selectAll {
		err = DB.Table("activity").First(&activity, "id = ?", id).Error
	} else {
		err = DB.Table("activity").Select([]string{"id", "activity_text as content", "activity_image as image"}).First(&activity, "id = ?", id).Error
	}
	return &activity, err
}
