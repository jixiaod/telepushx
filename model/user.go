package model

import (
	"errors"
)

type User struct {
	Id     int
	Name   string
	ChatId string
	Status int
}

func GetAllUsers(startIdx int, num int) (users []*User, err error) {
	err = DB.Table("users").Order("id asc").Limit(num).Offset(startIdx).Select([]string{"id", "name", "tete_id as chat_id", "status"}).Find(&users).Error
	return users, err
}

func GetMaxUserId() int {
	var user User
	DB.Last(&user)
	return user.Id
}

func GetUserById(id int, selectAll bool) (*User, error) {
	if id == 0 {
		return nil, errors.New("id 为空！")
	}
	user := User{Id: id}
	var err error = nil
	if selectAll {
		err = DB.Table("users").First(&user, "id = ?", id).Error
	} else {
		err = DB.Table("users").Select([]string{"id", "name", "tete_id as chat_id", "status"}).First(&user, "id = ?", id).Error
	}
	return &user, err
}
