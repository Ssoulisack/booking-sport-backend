package migration

import (
	"backend/internal/core/domain"
	"log"

	"gorm.io/gorm"
)

// RunMigrations runs all database migrations using GORM
func DatabaseMigrations(db *gorm.DB) error {
	// Add all domain and repository models here
	if err := db.AutoMigrate(
		&domain.Role{},
		&domain.User{},
		&domain.Sport{},
		&domain.Resource{},
		&domain.Coach{},
		&domain.OpeningHour{},
		&domain.Closure{},
		&domain.Price{},
		&domain.ClassSession{},
		&domain.BookingRule{},
	); err != nil {
		return err
	}
	log.Println("🌟 GORM database migrations completed successfully 🚀")
	return SetupInitialData(db)
}
