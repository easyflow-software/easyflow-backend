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

	return &DatabaseInst{client: db}, nil
}

func (d *DatabaseInst) GetClient() *gorm.DB {
	return d.client
}

func (d *DatabaseInst) Migrate() error {
	return d.client.AutoMigrate(&Message{}, &Chat{}, &User{}, &ChatUserKeys{}, &UserKeys{})
}

func (d *DatabaseInst) SetLogMode(mode logger.LogLevel) {
	d.client.Logger.LogMode(mode)
}
