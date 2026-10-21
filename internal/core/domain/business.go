package domain

import (
	"time"

	"gorm.io/gorm"
)

// PlatformRoleSuperAdmin is the only platform-level role; everyone else has an empty platform role
const PlatformRoleSuperAdmin = "super_admin"

// Business member roles, from most to least powerful
const (
	MemberOwner   = "owner"
	MemberManager = "manager"
	MemberStaff   = "staff"
)

// MemberRank orders member roles: a role can do everything a lower-ranked role can
func MemberRank(role string) int {
	switch role {
	case MemberOwner:
		return 3
	case MemberManager:
		return 2
	case MemberStaff:
		return 1
	}
	return 0
}

// Business status as shown to users; stored as the Published and Suspended flags
const (
	StatusDraft     = "draft"
	StatusPublished = "published"
	StatusSuspended = "suspended"
)

// How customers may book at a business
const (
	BookingModeAccount = "account"
	BookingModeGuest   = "guest"
	BookingModeBoth    = "both"
)

// PlanLimits are the maximum counts a business may create
type PlanLimits struct {
	MaxSports                int `json:"max_sports"`
	MaxResources             int `json:"max_resources"`
	MaxCoaches               int `json:"max_coaches"`
	MaxClassSessionsPerMonth int `json:"max_class_sessions_per_month"`
	MaxMembers               int `json:"max_members"`
	MaxStorageMB             int `json:"max_storage_mb"`
	MaxImages                int `json:"max_images"`
}

// LimitOverrides replace single plan limits for one business; nil keeps the plan value
type LimitOverrides struct {
	MaxSports                *int `json:"max_sports"`
	MaxResources             *int `json:"max_resources"`
	MaxCoaches               *int `json:"max_coaches"`
	MaxClassSessionsPerMonth *int `json:"max_class_sessions_per_month"`
	MaxMembers               *int `json:"max_members"`
	MaxStorageMB             *int `json:"max_storage_mb"`
	MaxImages                *int `json:"max_images"`
}

type Plan struct {
	gorm.Model
	Name   string     `gorm:"type:varchar(50);not null;uniqueIndex:idx_plans_name,where:deleted_at IS NULL"`
	Limits PlanLimits `gorm:"embedded"`
}

// Business is a tenant: a sports business with its own booking website
type Business struct {
	gorm.Model
	Name        string         `gorm:"type:varchar(120);not null"`
	Slug        string         `gorm:"type:varchar(60);not null;uniqueIndex:idx_businesses_slug,where:deleted_at IS NULL"`
	PlanID      uint           `gorm:"not null;index"`
	Plan        Plan           `gorm:"foreignKey:PlanID"`
	Overrides   LimitOverrides `gorm:"embedded;embeddedPrefix:override_"`
	Published   bool           `gorm:"not null;default:false"`
	PublishedAt *time.Time     // first publish; the slug cannot change after it
	Suspended   bool           `gorm:"not null;default:false"`

	// settings
	Timezone      string            `gorm:"type:varchar(64);not null;default:Asia/Bangkok"`
	Currency      string            `gorm:"type:varchar(3);not null;default:LAK"`
	Language      string            `gorm:"type:varchar(5);not null;default:en"`
	LogoKey       string            `gorm:"type:varchar(255)"`
	CoverKey      string            `gorm:"type:varchar(255)"`
	BrandColor    string            `gorm:"type:varchar(7)"`
	Description   string            `gorm:"type:text"`
	Address       string            `gorm:"type:varchar(255)"`
	Phone         string            `gorm:"type:varchar(30)"`
	MapURL        string            `gorm:"type:varchar(500)"`
	SocialLinks   map[string]string `gorm:"type:jsonb;serializer:json"`
	BookingPolicy string            `gorm:"type:text"`
	BookingMode   string            `gorm:"type:varchar(10);not null;default:account"`
}

func (b Business) Status() string {
	switch {
	case b.Suspended:
		return StatusSuspended
	case b.Published:
		return StatusPublished
	}
	return StatusDraft
}

// IsPublic tells whether the public booking page may be shown
func (b Business) IsPublic() bool { return b.Published && !b.Suspended }

// EffectiveLimits merges the plan limits with this business's overrides. Plan must be loaded.
func (b Business) EffectiveLimits() PlanLimits {
	l := b.Plan.Limits
	o := b.Overrides
	pick := func(dst *int, src *int) {
		if src != nil {
			*dst = *src
		}
	}
	pick(&l.MaxSports, o.MaxSports)
	pick(&l.MaxResources, o.MaxResources)
	pick(&l.MaxCoaches, o.MaxCoaches)
	pick(&l.MaxClassSessionsPerMonth, o.MaxClassSessionsPerMonth)
	pick(&l.MaxMembers, o.MaxMembers)
	pick(&l.MaxStorageMB, o.MaxStorageMB)
	pick(&l.MaxImages, o.MaxImages)
	return l
}

// Membership gives a user a role in a business. Removed members are deleted, not soft-deleted.
type Membership struct {
	ID         uint `gorm:"primaryKey"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
	UserID     uint     `gorm:"not null;uniqueIndex:idx_memberships_user_business,priority:1"`
	BusinessID uint     `gorm:"not null;uniqueIndex:idx_memberships_user_business,priority:2;index"`
	Role       string   `gorm:"type:varchar(10);not null"`
	Active     bool     `gorm:"not null;default:true"`
	User       User     `gorm:"foreignKey:UserID"`
	Business   Business `gorm:"foreignKey:BusinessID"`
}

// Invitation is a pending email invite to join a business. Only the token's hash is stored.
type Invitation struct {
	ID          uint `gorm:"primaryKey"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	BusinessID  uint      `gorm:"not null;index"`
	Email       string    `gorm:"type:varchar(255);not null;index"`
	Role        string    `gorm:"type:varchar(10);not null"`
	TokenHash   string    `gorm:"type:varchar(64);not null;uniqueIndex"`
	ExpiresAt   time.Time `gorm:"not null"`
	AcceptedAt  *time.Time
	InvitedByID uint `gorm:"not null"`
}

// PasswordReset is a single-use reset link. Only the token's hash is stored.
type PasswordReset struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	UserID    uint      `gorm:"not null;index"`
	TokenHash string    `gorm:"type:varchar(64);not null;uniqueIndex"`
	ExpiresAt time.Time `gorm:"not null"`
	UsedAt    *time.Time
}

// Media records an uploaded file so storage use can be limited per business
type Media struct {
	ID         uint `gorm:"primaryKey"`
	CreatedAt  time.Time
	BusinessID uint   `gorm:"not null;index"`
	Key        string `gorm:"type:varchar(255);not null;uniqueIndex"`
	SizeBytes  int64  `gorm:"not null"`
}

// SupportVisit records a Super Admin viewing a business dashboard
type SupportVisit struct {
	ID         uint      `gorm:"primaryKey"`
	UserID     uint      `gorm:"not null"`
	BusinessID uint      `gorm:"not null;index"`
	VisitedAt  time.Time `gorm:"not null;index"`
}
