package service

import (
	"errors"

	"gorm.io/gorm"
)

func isMySQLDialector(tx *gorm.DB) bool {
	return tx != nil && tx.Dialector != nil && tx.Dialector.Name() == "mysql"
}

func acquireNamedMySQLLock(tx *gorm.DB, name string) (bool, error) {
	if !isMySQLDialector(tx) || name == "" {
		return false, nil
	}
	var got int
	if err := tx.Session(&gorm.Session{NewDB: true}).Raw("SELECT GET_LOCK(?, 10)", name).Scan(&got).Error; err != nil {
		return false, err
	}
	if got != 1 {
		return false, errors.New("could not acquire award lock")
	}
	return true, nil
}

func releaseNamedMySQLLock(tx *gorm.DB, name string, locked bool) {
	if !locked || name == "" || tx == nil {
		return
	}
	_ = tx.Session(&gorm.Session{NewDB: true}).Exec("SELECT RELEASE_LOCK(?)", name).Error
}
