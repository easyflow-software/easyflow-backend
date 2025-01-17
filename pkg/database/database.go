package database

import (
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type DatabaseInst struct {
	client *gorm.DB
}

func NewDatabaseInst(url string, config *gorm.Config) (*DatabaseInst, error) {
	db, err := gorm.Open(mysql.Open(url), config)
	if err != nil {
		return nil, err
	}

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

func (d *DatabaseInst) SetLogMode(mode logger.LogLevel) {
	d.client.Logger.LogMode(mode)
}
