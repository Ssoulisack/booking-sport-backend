package output

import (
	"backend/internal/core/domain"
	"context"
)

// DirectoryQuery filters the public list of places; empty fields do not filter
type DirectoryQuery struct {
	Page      int
	Limit     int
	Keyword   string // name or address
	SportType string // has an active sport of this type
	Province  string
}

// DirectoryRepository reads the public list of places to play ("Find a place")
type DirectoryRepository interface {
	// Businesses lists published, not suspended businesses, newest published first
	Businesses(ctx context.Context, q DirectoryQuery) ([]domain.Business, int64, error)
	// ActiveSports returns the active sports of each business, by business id
	ActiveSports(ctx context.Context, businessIDs []uint) (map[uint][]domain.Sport, error)
	// LowestCourtPrices returns the lowest hourly price of an active court per business (none: absent)
	LowestCourtPrices(ctx context.Context, businessIDs []uint) (map[uint]domain.Price, error)
}
