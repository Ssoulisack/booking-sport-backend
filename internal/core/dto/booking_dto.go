package dto

import (
	"backend/internal/core/domain"
	"time"
)

// AvailabilityQuery asks for free starts on one date (YYYY-MM-DD, business timezone) for a
// resource, a coach, or both together. sport_id is needed when only a coach is given.
type AvailabilityQuery struct {
	Date       string `query:"date"`
	ResourceID uint   `query:"resource_id"`
	CoachID    uint   `query:"coach_id"`
	SportID    uint   `query:"sport_id"`
}

type SlotDTO struct {
	StartsAt  time.Time `json:"starts_at"`
	Durations []int     `json:"durations_min"`
}

type AvailabilityDTO struct {
	Date     string               `json:"date"`
	Timezone string               `json:"timezone"`
	Rule     domain.EffectiveRule `json:"rule"`
	Slots    []SlotDTO            `json:"slots"`
}

type GuestInfo struct {
	Name  string `json:"name"`
	Phone string `json:"phone"`
	Email string `json:"email"`
}

// CreateBookingRequest books a resource and/or coach (starts_at + duration_min), or seats in a
// group class (class_session_id + seats). Guests fill guest; logged-in customers leave it empty.
type CreateBookingRequest struct {
	ResourceID     *uint      `json:"resource_id"`
	CoachID        *uint      `json:"coach_id"`
	SportID        uint       `json:"sport_id"`
	StartsAt       time.Time  `json:"starts_at"`
	DurationMin    int        `json:"duration_min"`
	ClassSessionID *uint      `json:"class_session_id"`
	Seats          int        `json:"seats"`
	Guest          *GuestInfo `json:"guest"`
	Notes          string     `json:"notes"`
}

type CancelBookingRequest struct {
	Reason string `json:"reason"`
}

type BookingItemDTO struct {
	Kind           string    `json:"kind"`
	ResourceID     *uint     `json:"resource_id,omitempty"`
	CoachID        *uint     `json:"coach_id,omitempty"`
	ClassSessionID *uint     `json:"class_session_id,omitempty"`
	Seats          int       `json:"seats"`
	StartsAt       time.Time `json:"starts_at"`
	EndsAt         time.Time `json:"ends_at"`
	Price          int64     `json:"price"`
}

type BookingDTO struct {
	ID               uint             `json:"id"`
	Code             string           `json:"code"`
	BusinessName     string           `json:"business_name"`
	BusinessSlug     string           `json:"business_slug"`
	Status           string           `json:"status"`
	StartsAt         time.Time        `json:"starts_at"`
	EndsAt           time.Time        `json:"ends_at"`
	TotalPrice       int64            `json:"total_price"`
	Currency         string           `json:"currency"`
	PaymentStatus    string           `json:"payment_status"`
	CancellableUntil time.Time        `json:"cancellable_until"`
	GuestName        string           `json:"guest_name,omitempty"`
	Notes            string           `json:"notes"`
	CancelledAt      *time.Time       `json:"cancelled_at,omitempty"`
	CancelReason     string           `json:"cancel_reason,omitempty"`
	CreatedAt        time.Time        `json:"created_at"`
	Items            []BookingItemDTO `json:"items"`
}

// CreatedBookingDTO adds the guest's private link, returned only once at booking time
type CreatedBookingDTO struct {
	BookingDTO
	GuestToken string `json:"guest_token,omitempty"`
	GuestLink  string `json:"guest_link,omitempty"`
}
