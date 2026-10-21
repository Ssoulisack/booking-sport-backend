package output

import (
	"backend/internal/core/domain"
	"context"
	"time"
)

type BookingRepository interface {
	// Create stores the booking with its items in one transaction. Items overlapping an
	// active item of the same resource or coach are refused by the database (409 SLOT_NOT_AVAILABLE).
	Create(ctx context.Context, b *domain.Booking) error
	// CreateClassBooking locks the class session, checks that seats are still free, then stores
	// the booking; two customers cannot take the last seat at the same time (409 CLASS_FULL)
	CreateClassBooking(ctx context.Context, b *domain.Booking, classID uint, seats int) error
	GetByID(ctx context.Context, businessID, id uint) (*domain.Booking, error)
	GetForCustomer(ctx context.Context, customerID, id uint) (*domain.Booking, error)
	// GetByGuestToken returns (nil, nil) when no booking of the business has that token
	GetByGuestToken(ctx context.Context, businessID uint, tokenHash string) (*domain.Booking, error)
	ListForCustomer(ctx context.Context, customerID uint, q ListQuery) ([]domain.Booking, int64, error)
	// Cancel marks the booking cancelled and frees its items, in one transaction
	Cancel(ctx context.Context, b *domain.Booking, reason string, at time.Time) error

	// ActiveItems lists active items of a resource or coach overlapping [from, to)
	ActiveItems(ctx context.Context, businessID uint, ownerType string, ownerID uint, from, to time.Time) ([]domain.BookingItem, error)
	// CountActive counts pending / confirmed bookings ending after now, by customer or by guest phone
	CountActiveForCustomer(ctx context.Context, businessID, customerID uint, now time.Time) (int64, error)
	CountActiveForGuestPhone(ctx context.Context, businessID uint, phone string, now time.Time) (int64, error)
	// SeatsTaken sums the active seats booked per class session
	SeatsTaken(ctx context.Context, businessID uint, classIDs []uint) (map[uint]int, error)
	// CountFutureForOwner counts active customer items of a resource or coach ending after now
	CountFutureForOwner(ctx context.Context, businessID uint, ownerType string, ownerID uint, now time.Time) (int64, error)
}
