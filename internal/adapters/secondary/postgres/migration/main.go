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
	); err != nil {
		return err
	}
	log.Println("🌟 GORM database migrations completed successfully 🚀")
	return SetupInitialData(db)
}
