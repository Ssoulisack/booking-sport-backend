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
		&domain.Booking{},
		&domain.BookingItem{},
	); err != nil {
		return err
	}
	if err := retireRoles(db); err != nil {
		return err
	}
	if err := noDoubleBooking(db); err != nil {
		return err
	}
	// rule added in phase 3: give existing default rules a value so they stay complete
	if err := db.Exec("UPDATE booking_rules SET max_active_bookings_per_customer = 3 WHERE scope_type = ? AND max_active_bookings_per_customer IS NULL",
		domain.ScopeDefault).Error; err != nil {
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

// noDoubleBooking adds the database rules that make two active items of the same resource, or of
// the same coach, with overlapping times impossible, even when requests arrive at the same moment.
// Safe to run on every start.
func noDoubleBooking(db *gorm.DB) error {
	if err := db.Exec("CREATE EXTENSION IF NOT EXISTS btree_gist").Error; err != nil {
		return err
	}
	constraints := []struct{ name, column string }{
		{"booking_items_resource_no_overlap", "resource_id"},
		{"booking_items_coach_no_overlap", "coach_id"},
	}
	for _, c := range constraints {
		var exists int64
		if err := db.Raw("SELECT COUNT(*) FROM pg_constraint WHERE conname = ?", c.name).Scan(&exists).Error; err != nil {
			return err
		}
		if exists > 0 {
			continue
		}
		sql := "ALTER TABLE booking_items ADD CONSTRAINT " + c.name + " EXCLUDE USING gist (" +
			c.column + " WITH =, tstzrange(starts_at, ends_at, '[)') WITH &&) WHERE (active AND " + c.column + " IS NOT NULL)"
		if err := db.Exec(sql).Error; err != nil {
			return err
		}
	}
	return nil
}
