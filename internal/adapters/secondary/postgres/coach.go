package postgres

import (
	"backend/internal/adapters/secondary/postgres/exception"
	"backend/internal/core/domain"
	"backend/internal/core/port/output"
	"backend/pkg/response"
	"context"
	"errors"
	"net/http"
	"time"

	"gorm.io/gorm"
)

type coachRepository struct {
	*scopedRepository[domain.Coach]
}

// NewCoachRepository creates a coach repository that always loads the coach's sports
func NewCoachRepository(db *gorm.DB) output.CoachRepository {
	return &coachRepository{newScoped[domain.Coach](db, "name", "Sports")}
}

func (r *coachRepository) SetSports(ctx context.Context, businessID, coachID uint, sportIDs []uint) error {
	if businessID == 0 {
		return errNoScope
	}
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var coach domain.Coach
		if err := tx.Where("business_id = ?", businessID).First(&coach, coachID).Error; err != nil {
			return err
		}
		var sports []domain.Sport
		if len(sportIDs) > 0 {
			// only this business's sports: an id from another business counts as not found
			if err := tx.Where("business_id = ?", businessID).Find(&sports, sportIDs).Error; err != nil {
				return err
			}
			if len(sports) != len(uniqueIDs(sportIDs)) {
				return response.NewError(http.StatusBadRequest, "SPORT_NOT_FOUND")
			}
		}
		return tx.Model(&coach).Association("Sports").Replace(sports)
	})
	return keepAppError(err)
}

// keepAppError passes our own AppErrors through and maps everything else
func keepAppError(err error) error {
	var appErr response.AppError
	if errors.As(err, &appErr) {
		return err
	}
	return exception.HandleGormError(err)
}

func uniqueIDs(ids []uint) map[uint]struct{} {
	set := make(map[uint]struct{}, len(ids))
	for _, id := range ids {
		set[id] = struct{}{}
	}
	return set
}

type classSessionRepository struct {
	*scopedRepository[domain.ClassSession]
}

func NewClassSessionRepository(db *gorm.DB) output.ClassSessionRepository {
	return &classSessionRepository{newScoped[domain.ClassSession](db, "title")}
}

func (r *classSessionRepository) CountStartingBetween(ctx context.Context, businessID uint, from, to time.Time) (int64, error) {
	tx, err := r.scope(ctx, businessID)
	if err != nil {
		return 0, err
	}
	var total int64
	err = tx.Where("starts_at >= ? AND starts_at < ?", from, to).Count(&total).Error
	return total, exception.HandleGormError(err)
}

func (r *classSessionRepository) ListUpcoming(ctx context.Context, businessID uint, from time.Time, q output.ListQuery) ([]domain.ClassSession, int64, error) {
	tx, err := r.scope(ctx, businessID)
	if err != nil {
		return nil, 0, err
	}
	tx = applyFilters(tx.Where("active = ? AND starts_at >= ?", true, from), q.Filters)
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, exception.HandleGormError(err)
	}
	var items []domain.ClassSession
	err = paginate(tx.Order("starts_at ASC"), q).Find(&items).Error
	return items, total, exception.HandleGormError(err)
}

// syncClassHold makes an active class occupy its court and coach like a booking would, so a rental
// can never overlap a class (the database refuses it). Inactive classes hold nothing.
func syncClassHold(tx *gorm.DB, m *domain.ClassSession) error {
	if err := tx.Where("class_session_id = ? AND kind = ?", m.ID, domain.ItemClassHold).Delete(&domain.BookingItem{}).Error; err != nil {
		return err
	}
	if !m.Active || (m.ResourceID == nil && m.CoachID == nil) {
		return nil
	}
	classID := m.ID
	hold := domain.BookingItem{
		BusinessID: m.BusinessID, Kind: domain.ItemClassHold, ResourceID: m.ResourceID, CoachID: m.CoachID,
		ClassSessionID: &classID, Seats: 0, StartsAt: m.StartsAt, EndsAt: m.EndsAt, Active: true,
	}
	return tx.Create(&hold).Error
}

func (r *classSessionRepository) Create(ctx context.Context, m *domain.ClassSession) error {
	return keepAppError(r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(m).Error; err != nil {
			return err
		}
		return syncClassHold(tx, m)
	}))
}

func (r *classSessionRepository) Update(ctx context.Context, m *domain.ClassSession) error {
	return keepAppError(r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(m).Error; err != nil {
			return err
		}
		return syncClassHold(tx, m)
	}))
}

func (r *classSessionRepository) Delete(ctx context.Context, businessID, id uint) error {
	if businessID == 0 {
		return errNoScope
	}
	return keepAppError(r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Where("business_id = ?", businessID).Delete(&domain.ClassSession{}, id)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return tx.Where("class_session_id = ? AND kind = ?", id, domain.ItemClassHold).Delete(&domain.BookingItem{}).Error
	}))
}
