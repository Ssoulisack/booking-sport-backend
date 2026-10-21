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
		&domain.User{},
		&domain.Plan{},
		&domain.Business{},
		&domain.Membership{},
		&domain.Invitation{},
		&domain.PasswordReset{},
		&domain.Media{},
		&domain.SupportVisit{},
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
	if err := retireRoles(db); err != nil {
		return err
	}
	log.Println("🌟 GORM database migrations completed successfully 🚀")
	return SetupInitialData(db)
}

// retireRoles converts a database from phase 1 (roles table + users.role_id) to platform roles:
// the old admin role (1) becomes the Super Admin. Safe to run on every start.
func retireRoles(db *gorm.DB) error {
	m := db.Migrator()
	if m.HasColumn(&domain.User{}, "role_id") {
		if err := db.Exec("UPDATE users SET platform_role = ? WHERE role_id = 1", domain.PlatformRoleSuperAdmin).Error; err != nil {
			return err
		}
		if err := m.DropColumn(&domain.User{}, "role_id"); err != nil {
			return err
		}
	}
	if m.HasTable("roles") {
		return m.DropTable("roles")
	}
	return nil
}
