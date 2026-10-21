package domain

import "gorm.io/gorm"

// Rule scopes, from widest to most specific
const (
	ScopeGlobal   = "global"
	ScopeSport    = "sport"
	ScopeResource = "resource"
)

// BookingRule holds editable booking rules. A nil field inherits from the next wider scope
// (resource -> sport -> global). The global rule must have every field set.
// ScopeID is 0 for the global rule.
type BookingRule struct {
	gorm.Model
	ScopeType         string `gorm:"type:varchar(20);not null;uniqueIndex:idx_rules_scope,priority:1,where:deleted_at IS NULL"`
	ScopeID           uint   `gorm:"not null;default:0;uniqueIndex:idx_rules_scope,priority:2,where:deleted_at IS NULL"`
	MinDurationMin    *int
	MaxDurationMin    *int
	DurationStepMin   *int
	StartStepMin      *int
	CancelBeforeHours *int
	MaxDaysAhead      *int
	MinNoticeMin      *int
	Capacity          *int // default participants for group sessions
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
