package migration

import (
	"backend/internal/core/domain"

	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func SetupInitialData(db *gorm.DB) error {
	encryptPassword, err := bcrypt.GenerateFromPassword([]byte("123456"), 14)
	if err != nil {
		return err
	}
	// Create users
	users := []domain.User{
		{
			Email:        "soulisack@gmail.com",
			Username:     "24thurs",
			Name:         "Soulisack DVL",
			Verified:     true,
			Status:       true,
			LastActive:   time.Now(),
			PasswordHash: encryptPassword,
			Phone:        "0123456789",
		},
		{
			Email:        "24thurs@gmail.com",
			Username:     "sack",
			Name:         "alex DVL",
			Verified:     true,
			Status:       true,
			LastActive:   time.Now(),
			PasswordHash: encryptPassword,
			Phone:        "9876543210",
		},
	}

	for i := range users {
		if err := db.Create(&users[i]).Error; err != nil {
			return err
		}
	}

	return nil
}
