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

	if err := seedGlobalRule(db); err != nil {
		return err
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

// seedGlobalRule creates the default booking rule once; staff edit it in the back office afterwards
func seedGlobalRule(db *gorm.DB) error {
	var count int64
	if err := db.Model(&domain.BookingRule{}).
		Where("scope_type = ? AND scope_id = 0", domain.ScopeGlobal).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	v := func(n int) *int { return &n }
	return db.Create(&domain.BookingRule{
		ScopeType:         domain.ScopeGlobal,
		MinDurationMin:    v(60),
		MaxDurationMin:    v(180),
		DurationStepMin:   v(30),
		StartStepMin:      v(30),
		CancelBeforeHours: v(24),
		MaxDaysAhead:      v(30),
		MinNoticeMin:      v(60),
		Capacity:          v(10),
	}).Error
}
