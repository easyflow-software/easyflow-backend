package database

import (
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type DatabaseInst struct {
	client *gorm.DB
}

func NewDatabaseInst(url string, config *gorm.Config) (*DatabaseInst, error) {
	db, err := gorm.Open(mysql.Open(url), config)
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(500)
	sqlDB.SetConnMaxLifetime(time.Hour)

	err = db.SetupJoinTable(&User{}, "Chats", &ChatsUsers{})
	if err != nil {
		panic(err)
	}
	err = db.SetupJoinTable(&Chat{}, "Users", &ChatsUsers{})
	if err != nil {
		panic(err)
	}

	return &DatabaseInst{client: db}, nil
}

func (d *DatabaseInst) GetClient() *gorm.DB {
	return d.client
}

func (d *DatabaseInst) Migrate() error {
	return d.client.AutoMigrate(&Message{}, &Chat{}, &User{}, &ChatsUsers{}, &UserKeys{})
}
