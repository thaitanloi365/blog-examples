package db

import (
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
)

type DB struct {
	*gorm.DB
}

func New(driver gorm.Dialector) *DB {
	db, err := gorm.Open(driver, &gorm.Config{
		Logger: gormLogger.Default.LogMode(gormLogger.Info),
	})
	if err != nil {
		panic(err)
	}

	return &DB{
		DB: db,
	}
}
func (db *DB) WithGorm(gdb *gorm.DB) *DB {
	return &DB{
		DB: gdb,
	}
}
