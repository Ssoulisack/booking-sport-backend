package postgres

import (
	"backend/internal/adapters/secondary/postgres/exception"
	"backend/internal/core/domain"
	"backend/internal/core/port/output"
	"context"
	"errors"
	"net/http"

	"backend/pkg/response"

	"gorm.io/gorm"
)

type coachRepository struct {
	*crudRepository[domain.Coach]
}

// NewCoachRepository creates a coach repository that always loads the coach's sports
func NewCoachRepository(db *gorm.DB) output.CoachRepository {
	return &coachRepository{newCRUD[domain.Coach](db, "name", "Sports")}
}

func (r *coachRepository) SetSports(ctx context.Context, coachID uint, sportIDs []uint) error {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var coach domain.Coach
		if err := tx.First(&coach, coachID).Error; err != nil {
			return err
		}
		var sports []domain.Sport
		if len(sportIDs) > 0 {
			if err := tx.Find(&sports, sportIDs).Error; err != nil {
				return err
			}
			if len(sports) != len(uniqueIDs(sportIDs)) {
				return response.NewError(http.StatusBadRequest, "SPORT_NOT_FOUND")
			}
		}
		return tx.Model(&coach).Association("Sports").Replace(sports)
	})
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
