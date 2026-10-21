package migration

import (
	"backend/internal/core/domain"
	"log"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// Default back-office admin, created once. Change the password after first login.
const (
	adminEmail    = "admin@booking.local"
	adminPassword = "ChangeMe123!"
)

// SetupInitialData is idempotent: it only creates what is missing
func SetupInitialData(db *gorm.DB) error {
	roles := []domain.Role{
		{ID: domain.RoleAdmin, Name: "admin"},
		{ID: domain.RoleStaff, Name: "staff"},
		{ID: domain.RoleClient, Name: "client"},
	}
	for i := range roles {
		if err := db.Where("id = ?", roles[i].ID).FirstOrCreate(&roles[i]).Error; err != nil {
			return err
		}
	}

	var count int64
	if err := db.Model(&domain.User{}).Where("email = ?", adminEmail).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(adminPassword), 14)
	if err != nil {
		return err
	}
	admin := domain.User{
		Email:        adminEmail,
		Username:     adminEmail,
		Name:         "Administrator",
		RoleID:       domain.RoleAdmin,
		Verified:     true,
		Status:       true,
		LastActive:   time.Now(),
		PasswordHash: hash,
	}
	if err := db.Create(&admin).Error; err != nil {
		return err
	}
	log.Printf("⚠️  Default admin created: %s — change its password", adminEmail)
	return nil
}
