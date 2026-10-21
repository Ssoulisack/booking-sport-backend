package domain

import "gorm.io/gorm"

// Rule scopes, from widest to most specific
const (
	ScopeDefault  = "default" // the business's default rule
	ScopeSport    = "sport"
	ScopeResource = "resource"
)

// BookingRule holds editable booking rules of one business. A nil field inherits from the next
// wider scope (resource -> sport -> default). The default rule must have every field set.
// ScopeID is 0 for the default rule.
type BookingRule struct {
	gorm.Model
	BusinessID        uint   `gorm:"not null;uniqueIndex:idx_rules_scope,priority:1,where:deleted_at IS NULL"`
	ScopeType         string `gorm:"type:varchar(20);not null;uniqueIndex:idx_rules_scope,priority:2,where:deleted_at IS NULL"`
	ScopeID           uint   `gorm:"not null;default:0;uniqueIndex:idx_rules_scope,priority:3,where:deleted_at IS NULL"`
	MinDurationMin    *int
	MaxDurationMin    *int
	DurationStepMin   *int
	StartStepMin      *int
	CancelBeforeHours *int
	MaxDaysAhead      *int
	MinNoticeMin      *int
	Capacity          *int // default participants for group sessions
}

// DefaultBookingRule is the rule every new business starts with; its staff can edit it
func DefaultBookingRule(businessID uint) BookingRule {
	v := func(n int) *int { return &n }
	return BookingRule{
		BusinessID:        businessID,
		ScopeType:         ScopeDefault,
		MinDurationMin:    v(60),
		MaxDurationMin:    v(180),
		DurationStepMin:   v(30),
		StartStepMin:      v(30),
		CancelBeforeHours: v(24),
		MaxDaysAhead:      v(30),
		MinNoticeMin:      v(60),
		Capacity:          v(10),
	}
}

// EffectiveRule is the result of merging rules; every field is set
type EffectiveRule struct {
	MinDurationMin    int `json:"min_duration_min"`
	MaxDurationMin    int `json:"max_duration_min"`
	DurationStepMin   int `json:"duration_step_min"`
	StartStepMin      int `json:"start_step_min"`
	CancelBeforeHours int `json:"cancel_before_hours"`
	MaxDaysAhead      int `json:"max_days_ahead"`
	MinNoticeMin      int `json:"min_notice_min"`
	Capacity          int `json:"capacity"`
}
