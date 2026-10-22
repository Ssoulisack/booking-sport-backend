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
	if err := onePhonePerAccount(db); err != nil {
		return err
	}
	if err := phonesAsTwentyDigits(db); err != nil {
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

// onePhonePerAccount (phase 6d.2) stores every phone number in one form (+8562055551234) and then makes
// a number belong to one account only, because people log in with it. Before the rule is added, a
// number found on several accounts stays on the oldest and is cleared from the others (logged).
// Safe to run on every start.
func onePhonePerAccount(db *gorm.DB) error {
	var exists int64
	if err := db.Raw("SELECT count(*) FROM pg_indexes WHERE indexname = 'idx_users_phone_unique'").Scan(&exists).Error; err != nil {
		return err
	}
	if exists > 0 {
		return nil
	}
	var users []domain.User
	if err := db.Where("phone <> ''").Order("id ASC").Find(&users).Error; err != nil { // live accounts, like the rule
		return err
	}
	seen := map[string]uint{}
	for _, u := range users {
		phone, ok := domain.NormalizePhone(u.Phone)
		if !ok {
			phone = u.Phone // keep what cannot be read; the owner can fix it in My account
		}
		if first, dup := seen[phone]; dup {
			log.Printf("⚠️  phone %s is on accounts %d and %d: kept on %d, cleared on %d", phone, first, u.ID, first, u.ID)
			phone = ""
		} else {
			seen[phone] = u.ID
		}
		if phone != u.Phone {
			if err := db.Exec("UPDATE users SET phone = ? WHERE id = ?", phone, u.ID).Error; err != nil {
				return err
			}
		}
	}
	return db.Exec("CREATE UNIQUE INDEX idx_users_phone_unique ON users (phone) WHERE phone <> '' AND deleted_at IS NULL").Error
}

// phonesAsTwentyDigits (phase 6d.2) moves stored account phones to the format people type and see,
// 20xxxxxxxx (e.g. "+8562055551234" becomes "2055551234"). Numbers that are not a 20 + 8 digit Lao
// mobile (030, landlines, other countries) are no longer valid for accounts and are cleared (logged).
// Safe to run on every start: it only touches phones not yet in that format.
func phonesAsTwentyDigits(db *gorm.DB) error {
	var users []domain.User
	if err := db.Where("phone <> '' AND phone !~ '^20[0-9]{8}$'").Order("id ASC").Find(&users).Error; err != nil {
		return err
	}
	for _, u := range users {
		phone, ok := domain.NormalizePhone(u.Phone)
		if ok {
			var taken int64
			if err := db.Model(&domain.User{}).Where("phone = ? AND id <> ?", phone, u.ID).Count(&taken).Error; err != nil {
				return err
			}
			ok = taken == 0
		}
		if !ok {
			log.Printf("⚠️  account %d: phone %q is not a 20xxxxxxxx Lao mobile (or is taken): cleared", u.ID, u.Phone)
			phone = ""
		}
		if err := db.Exec("UPDATE users SET phone = ? WHERE id = ?", phone, u.ID).Error; err != nil {
			return err
		}
	}
	return nil
}
