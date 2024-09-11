package model

import (
	"os"
	"telepushx/common"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var DB *gorm.DB

// Function to initialize the database connection
func InitDB() (err error) {
	var db *gorm.DB
	dbConnectionString := os.Getenv("MYSQL_CONN_STRING")

	db, err = gorm.Open(mysql.Open(dbConnectionString), &gorm.Config{})
	// Create the database connection string

	if err != nil {
		return err
	}

	if err == nil {
		DB = db
		err := db.AutoMigrate(&User{})
		if err != nil {
			return err
		}

		return err
	} else {
		common.FatalLog(err)
	}
	return err
}

func CloseDB() error {
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}
	err = sqlDB.Close()
	return err
}
