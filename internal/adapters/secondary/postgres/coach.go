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
