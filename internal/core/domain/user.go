package domain

import (
	"time"

	"gorm.io/gorm"
)

// User is the gorm model for the users table
type User struct {
	gorm.Model
	Email        string `gorm:"type:varchar(255);uniqueIndex"`
	Username     string `gorm:"type:varchar(255);uniqueIndex"`
	PasswordHash []byte
	Name         string `gorm:"type:varchar(255);not null"`
	Avatar       string `gorm:"type:varchar(255)"`
	Phone        string `gorm:"type:varchar(20)"`
	Verified     bool   `gorm:"type:boolean;not null;default:false"`
	Status       bool   `gorm:"type:boolean;not null;default:true"`
	LastActive   time.Time
}