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
	err = DB.Where("status = 1 AND tete_id > 0").Order("push_order desc, lastlog desc").Limit(num).Offset(startIdx).Select([]string{"id", "name", "tete_id", "status"}).Find(&users).Error
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

func GetActiveUserCount() (int64, error) {
	var count int64
	err := DB.Model(&User{}).Where("status = ?", 1).Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}

func UpdateUserStatusById(userId int, status int) error {
	if userId == 0 {
		return errors.New("userId 为空！")
	}

	return DB.Model(&User{}).Where("id = ?", userId).UpdateColumn("status", status).Error
}
