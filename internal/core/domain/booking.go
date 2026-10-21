package domain

import (
	"time"

	"gorm.io/gorm"
)

// Booking statuses. Pending and confirmed bookings hold their time; cancelled ones free it.
const (
	BookingPending   = "pending"
	BookingConfirmed = "confirmed"
	BookingCancelled = "cancelled"
	BookingCompleted = "completed"
	BookingNoShow    = "no_show"
)

// Booking item kinds. A class hold is not part of a customer booking: it makes a group class
// occupy its court and coach so nobody can rent them at the same time.
const (
	ItemResource     = "resource"
	ItemCoach        = "coach"
	ItemClassSession = "class_session"
	ItemClassHold    = "class_hold"
)

// Who created a booking
const (
	CreatedByCustomer = "customer"
	CreatedByGuest    = "guest"
	CreatedByStaff    = "staff"
)

const PaymentUnpaid = "unpaid"

// Booking is one customer's or guest's reservation at a business
type Booking struct {
	gorm.Model
	BusinessID uint     `gorm:"not null;index"`
	Business   Business `gorm:"foreignKey:BusinessID"`
	Code       string   `gorm:"type:varchar(12);not null;uniqueIndex"` // short reference shown to people

	CustomerID     *uint  `gorm:"index"` // nil for guests
	GuestName      string `gorm:"type:varchar(120)"`
	GuestPhone     string `gorm:"type:varchar(30);index"`
	GuestEmail     string `gorm:"type:varchar(255)"`
	GuestTokenHash string `gorm:"type:varchar(64);index"` // private link for guests; only the hash is stored

	Status           string    `gorm:"type:varchar(12);not null;index"`
	StartsAt         time.Time `gorm:"not null;index"`
	EndsAt           time.Time `gorm:"not null"`
	TotalPrice       int64     `gorm:"not null"`
	Currency         string    `gorm:"type:varchar(3);not null"`
	PaymentStatus    string    `gorm:"type:varchar(12);not null;default:unpaid"`
	CancellableUntil time.Time `gorm:"not null"` // rule snapshot: later rule changes do not affect this booking
	Notes            string    `gorm:"type:text"`
	CreatedByType    string    `gorm:"type:varchar(10);not null"`
	CancelledAt      *time.Time
	CancelReason     string `gorm:"type:varchar(255)"`

	Items []BookingItem `gorm:"foreignKey:BookingID"`
}

// IsActive tells whether the booking still holds its time
func (b Booking) IsActive() bool {
	return b.Status == BookingPending || b.Status == BookingConfirmed
}

// BookingItem is one thing a booking (or a class hold) occupies. Active items of the same
// resource or the same coach can never overlap: the database enforces it.
type BookingItem struct {
	ID             uint `gorm:"primaryKey"`
	CreatedAt      time.Time
	BookingID      *uint     `gorm:"index"` // nil for class holds
	BusinessID     uint      `gorm:"not null;index"`
	Kind           string    `gorm:"type:varchar(15);not null"`
	ResourceID     *uint     `gorm:"index"`
	CoachID        *uint     `gorm:"index"`
	ClassSessionID *uint     `gorm:"index"`
	Seats          int       `gorm:"not null;default:1"`
	StartsAt       time.Time `gorm:"not null"`
	EndsAt         time.Time `gorm:"not null"`
	Price          int64     `gorm:"not null;default:0"`
	Active         bool      `gorm:"not null;default:true"`
}
