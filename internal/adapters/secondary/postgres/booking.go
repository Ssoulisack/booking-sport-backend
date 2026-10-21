package postgres

import (
	"backend/internal/adapters/secondary/postgres/exception"
	"backend/internal/core/domain"
	"backend/internal/core/port/output"
	"backend/pkg/response"
	"context"
	"net/http"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var activeStatuses = []string{domain.BookingPending, domain.BookingConfirmed}

type bookingRepository struct{ db *gorm.DB }

func NewBookingRepository(db *gorm.DB) output.BookingRepository {
	return &bookingRepository{db: db}
}

func (r *bookingRepository) Create(ctx context.Context, b *domain.Booking) error {
	if b.BusinessID == 0 {
		return errNoScope
	}
	return exception.HandleGormError(r.db.WithContext(ctx).Omit("Business").Create(b).Error)
}

func (r *bookingRepository) CreateClassBooking(ctx context.Context, b *domain.Booking, classID uint, seats int) error {
	if b.BusinessID == 0 {
		return errNoScope
	}
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// the row lock makes concurrent bookings of the same class wait for each other
		var class domain.ClassSession
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("business_id = ?", b.BusinessID).First(&class, classID).Error; err != nil {
			return err
		}
		var taken int64
		if err := tx.Model(&domain.BookingItem{}).
			Where("class_session_id = ? AND kind = ? AND active", classID, domain.ItemClassSession).
			Select("COALESCE(SUM(seats), 0)").Scan(&taken).Error; err != nil {
			return err
		}
		if taken+int64(seats) > int64(class.Capacity) {
			return response.NewError(http.StatusConflict, "CLASS_FULL")
		}
		return tx.Omit("Business").Create(b).Error
	})
	return keepAppError(err)
}

func (r *bookingRepository) withDetails(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx).Preload("Items").Preload("Business")
}

func (r *bookingRepository) GetByID(ctx context.Context, businessID, id uint) (*domain.Booking, error) {
	if businessID == 0 {
		return nil, errNoScope
	}
	var b domain.Booking
	if err := r.withDetails(ctx).Where("business_id = ?", businessID).First(&b, id).Error; err != nil {
		return nil, exception.HandleGormError(err)
	}
	return &b, nil
}

func (r *bookingRepository) GetForCustomer(ctx context.Context, customerID, id uint) (*domain.Booking, error) {
	var b domain.Booking
	if err := r.withDetails(ctx).Where("customer_id = ?", customerID).First(&b, id).Error; err != nil {
		return nil, exception.HandleGormError(err)
	}
	return &b, nil
}

func (r *bookingRepository) GetByGuestToken(ctx context.Context, businessID uint, tokenHash string) (*domain.Booking, error) {
	if businessID == 0 {
		return nil, errNoScope
	}
	return firstOrNil[domain.Booking](r.withDetails(ctx).
		Where("business_id = ? AND guest_token_hash = ? AND guest_token_hash <> ''", businessID, tokenHash))
}

func (r *bookingRepository) ListForCustomer(ctx context.Context, customerID uint, q output.ListQuery) ([]domain.Booking, int64, error) {
	tx := applyFilters(r.db.WithContext(ctx).Model(&domain.Booking{}).Where("customer_id = ?", customerID), q.Filters)
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, exception.HandleGormError(err)
	}
	var items []domain.Booking
	err := paginate(tx.Preload("Items").Preload("Business").Order("starts_at DESC"), q).Find(&items).Error
	return items, total, exception.HandleGormError(err)
}

func (r *bookingRepository) Cancel(ctx context.Context, b *domain.Booking, reason string, at time.Time) error {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&domain.Booking{}).
			Where("id = ? AND business_id = ? AND status IN ?", b.ID, b.BusinessID, activeStatuses).
			Updates(map[string]any{"status": domain.BookingCancelled, "cancelled_at": at, "cancel_reason": reason})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected != 1 {
			return response.NewError(http.StatusConflict, "BOOKING_NOT_ACTIVE")
		}
		return tx.Model(&domain.BookingItem{}).Where("booking_id = ?", b.ID).Update("active", false).Error
	})
	if err != nil {
		return keepAppError(err)
	}
	b.Status, b.CancelledAt, b.CancelReason = domain.BookingCancelled, &at, reason
	return nil
}

// ownerColumn maps an owner type to its booking_items column
func ownerColumn(ownerType string) (string, error) {
	switch ownerType {
	case domain.OwnerResource:
		return "resource_id", nil
	case domain.OwnerCoach:
		return "coach_id", nil
	}
	return "", response.NewError(http.StatusBadRequest, "INVALID_OWNER")
}

func (r *bookingRepository) ActiveItems(ctx context.Context, businessID uint, ownerType string, ownerID uint, from, to time.Time) ([]domain.BookingItem, error) {
	if businessID == 0 {
		return nil, errNoScope
	}
	column, err := ownerColumn(ownerType)
	if err != nil {
		return nil, err
	}
	var items []domain.BookingItem
	err = r.db.WithContext(ctx).
		Where("business_id = ? AND active AND "+column+" = ? AND starts_at < ? AND ends_at > ?", businessID, ownerID, to, from).
		Order("starts_at ASC").Find(&items).Error
	return items, exception.HandleGormError(err)
}

func (r *bookingRepository) countActive(ctx context.Context, where string, args ...any) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&domain.Booking{}).Where(where, args...).Count(&count).Error
	return count, exception.HandleGormError(err)
}

func (r *bookingRepository) CountActiveForCustomer(ctx context.Context, businessID, customerID uint, now time.Time) (int64, error) {
	return r.countActive(ctx, "business_id = ? AND customer_id = ? AND status IN ? AND ends_at > ?",
		businessID, customerID, activeStatuses, now)
}

func (r *bookingRepository) CountActiveForGuestPhone(ctx context.Context, businessID uint, phone string, now time.Time) (int64, error) {
	return r.countActive(ctx, "business_id = ? AND customer_id IS NULL AND guest_phone = ? AND status IN ? AND ends_at > ?",
		businessID, phone, activeStatuses, now)
}

func (r *bookingRepository) SeatsTaken(ctx context.Context, businessID uint, classIDs []uint) (map[uint]int, error) {
	out := make(map[uint]int, len(classIDs))
	if len(classIDs) == 0 {
		return out, nil
	}
	var rows []struct {
		ClassSessionID uint
		Seats          int
	}
	err := r.db.WithContext(ctx).Model(&domain.BookingItem{}).
		Select("class_session_id, COALESCE(SUM(seats), 0) AS seats").
		Where("business_id = ? AND kind = ? AND active AND class_session_id IN ?", businessID, domain.ItemClassSession, classIDs).
		Group("class_session_id").Scan(&rows).Error
	if err != nil {
		return nil, exception.HandleGormError(err)
	}
	for _, row := range rows {
		out[row.ClassSessionID] = row.Seats
	}
	return out, nil
}

func (r *bookingRepository) CountFutureForOwner(ctx context.Context, businessID uint, ownerType string, ownerID uint, now time.Time) (int64, error) {
	column, err := ownerColumn(ownerType)
	if err != nil {
		return 0, err
	}
	var count int64
	err = r.db.WithContext(ctx).Model(&domain.BookingItem{}).
		Where("business_id = ? AND active AND kind <> ? AND "+column+" = ? AND ends_at > ?", businessID, domain.ItemClassHold, ownerID, now).
		Count(&count).Error
	return count, exception.HandleGormError(err)
}

// ---------- closures ----------

type closureRepository struct {
	*scopedRepository[domain.Closure]
}

func NewClosureRepository(db *gorm.DB) output.ClosureRepository {
	return &closureRepository{newScoped[domain.Closure](db, "")}
}

func (r *closureRepository) Overlapping(ctx context.Context, businessID uint, from, to time.Time) ([]domain.Closure, error) {
	tx, err := r.scope(ctx, businessID)
	if err != nil {
		return nil, err
	}
	var items []domain.Closure
	err = tx.Where("starts_at < ? AND ends_at > ?", to, from).Find(&items).Error
	return items, exception.HandleGormError(err)
}
