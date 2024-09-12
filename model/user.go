package model

import (
	"errors"
)

type User struct {
	Id        uint64 `gorm:"column:id;primaryKey;autoIncrement;type:bigint(20) unsigned"`
	Name      string `gorm:"column:name;type:varchar(255)"`
	Username  string `gorm:"column:username;type:varchar(255)"`
	ChatId    string `gorm:"column:tete_id;type:varchar(60)"`
	Status    int    `gorm:"column:status;type:int(11)"`
	PushOrder int    `gorm:"column:push_order;type:int(11)"`
	CreatedAt int64  `gorm:"column:created_at;type:timestamp"`
	UpdatedAt int64  `gorm:"column:updated_at;type:timestamp"`
}

func (User) TableName() string {
	return "users"
}

func GetAllUsers(startIdx int, num int) (users []*User, err error) {

	err = DB.Where("status = 1").Order("push_order desc").Limit(num).Offset(startIdx).Select([]string{"id", "name", "tete_id", "status"}).Find(&users).Error
	return users, err
}

func GetMaxUserId() int64 {
	var user User
	DB.Last(&user)
	return int64(user.Id)
}

func GetUserById(id int, selectAll bool) (*User, error) {
	if id == 0 {
		return nil, errors.New("id 为空！")
	}
	user := User{Id: uint64(id)}
	var err error = nil
	if selectAll {
		err = DB.First(&user, "id = ?", id).Error
	} else {
		err = DB.Select([]string{"id", "name", "tete_id", "status"}).First(&user, "id = ?", id).Error
	}
	return &user, err
}
