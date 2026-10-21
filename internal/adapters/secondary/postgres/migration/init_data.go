package migration

import (
	"backend/internal/core/domain"
	"log"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// Default Super Admin, created once. Change the password after first login.
const (
	adminEmail    = "admin@booking.local"
	adminPassword = "ChangeMe123!"
)

// defaultPlans are created once; the Super Admin edits the limits afterwards
var defaultPlans = []domain.Plan{
	{Name: "Free", Limits: domain.PlanLimits{
		MaxSports: 2, MaxResources: 3, MaxCoaches: 3, MaxClassSessionsPerMonth: 20,
		MaxMembers: 2, MaxStorageMB: 50, MaxImages: 100,
	}},
	{Name: "Basic", Limits: domain.PlanLimits{
		MaxSports: 5, MaxResources: 10, MaxCoaches: 10, MaxClassSessionsPerMonth: 100,
		MaxMembers: 5, MaxStorageMB: 200, MaxImages: 500,
	}},
	{Name: "Pro", Limits: domain.PlanLimits{
		MaxSports: 20, MaxResources: 50, MaxCoaches: 50, MaxClassSessionsPerMonth: 500,
		MaxMembers: 20, MaxStorageMB: 1024, MaxImages: 2000,
	}},
}

// SetupInitialData is idempotent: it only creates what is missing
func SetupInitialData(db *gorm.DB) error {
	for _, p := range defaultPlans {
		plan := p
		if err := db.Where("name = ?", plan.Name).FirstOrCreate(&plan).Error; err != nil {
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
		Name:         "Super Admin",
		PlatformRole: domain.PlatformRoleSuperAdmin,
		Verified:     true,
		Status:       true,
		LastActive:   time.Now(),
		PasswordHash: hash,
	}
	if err := db.Create(&admin).Error; err != nil {
		return err
	}
	log.Printf("⚠️  Default Super Admin created: %s — change its password", adminEmail)
	return nil
}
