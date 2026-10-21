package postgres

import (
	"backend/internal/adapters/secondary/postgres/exception"
	"backend/internal/core/domain"
	"backend/internal/core/port/output"
	"context"
	"errors"

	"gorm.io/gorm"
)

type openingHourRepository struct{ db *gorm.DB }

func NewOpeningHourRepository(db *gorm.DB) output.OpeningHourRepository {
	return &openingHourRepository{db: db}
}

func (r *openingHourRepository) ListByOwner(ctx context.Context, ownerType string, ownerID uint) ([]domain.OpeningHour, error) {
	var hours []domain.OpeningHour
	err := r.db.WithContext(ctx).
		Where("owner_type = ? AND owner_id = ?", ownerType, ownerID).
		Order("weekday ASC, open_time ASC").Find(&hours).Error
	return hours, exception.HandleGormError(err)
}

// ReplaceForOwner hard-deletes the old schedule: hours have no history worth keeping
func (r *openingHourRepository) ReplaceForOwner(ctx context.Context, ownerType string, ownerID uint, hours []domain.OpeningHour) error {
	return exception.HandleGormError(r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Unscoped().
			Where("owner_type = ? AND owner_id = ?", ownerType, ownerID).
			Delete(&domain.OpeningHour{}).Error; err != nil {
			return err
		}
		if len(hours) == 0 {
			return nil
		}
		return tx.Create(&hours).Error
	}))
}

type priceRepository struct{ db *gorm.DB }

func NewPriceRepository(db *gorm.DB) output.PriceRepository {
	return &priceRepository{db: db}
}

func (r *priceRepository) GetByOwner(ctx context.Context, ownerType string, ownerID uint) (*domain.Price, error) {
	var p domain.Price
	err := r.db.WithContext(ctx).Where("owner_type = ? AND owner_id = ?", ownerType, ownerID).First(&p).Error
	if err != nil {
		return nil, exception.HandleGormError(err)
	}
	return &p, nil
}

func (r *priceRepository) Upsert(ctx context.Context, p *domain.Price) error {
	return exception.HandleGormError(r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing domain.Price
		err := tx.Where("owner_type = ? AND owner_id = ?", p.OwnerType, p.OwnerID).First(&existing).Error
		switch {
		case err == nil:
			p.ID = existing.ID
			p.CreatedAt = existing.CreatedAt
			return tx.Save(p).Error
		case errors.Is(err, gorm.ErrRecordNotFound):
			return tx.Create(p).Error
		default:
			return err
		}
	}))
}

func (r *priceRepository) DeleteByOwner(ctx context.Context, ownerType string, ownerID uint) error {
	res := r.db.WithContext(ctx).Where("owner_type = ? AND owner_id = ?", ownerType, ownerID).Delete(&domain.Price{})
	if res.Error != nil {
		return exception.HandleGormError(res.Error)
	}
	if res.RowsAffected == 0 {
		return exception.HandleGormError(gorm.ErrRecordNotFound)
	}
	return nil
}

type ruleRepository struct{ db *gorm.DB }

func NewRuleRepository(db *gorm.DB) output.RuleRepository {
	return &ruleRepository{db: db}
}

func (r *ruleRepository) GetByScope(ctx context.Context, scopeType string, scopeID uint) (*domain.BookingRule, error) {
	var rule domain.BookingRule
	err := r.db.WithContext(ctx).Where("scope_type = ? AND scope_id = ?", scopeType, scopeID).First(&rule).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, exception.HandleGormError(err)
	}
	return &rule, nil
}

// Upsert replaces the whole rule, so a nil field is stored as NULL (inherit)
func (r *ruleRepository) Upsert(ctx context.Context, rule *domain.BookingRule) error {
	return exception.HandleGormError(r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing domain.BookingRule
		err := tx.Where("scope_type = ? AND scope_id = ?", rule.ScopeType, rule.ScopeID).First(&existing).Error
		switch {
		case err == nil:
			rule.ID = existing.ID
			rule.CreatedAt = existing.CreatedAt
			return tx.Save(rule).Error
		case errors.Is(err, gorm.ErrRecordNotFound):
			return tx.Create(rule).Error
		default:
			return err
		}
	}))
}

func (r *ruleRepository) List(ctx context.Context) ([]domain.BookingRule, error) {
	var rules []domain.BookingRule
	err := r.db.WithContext(ctx).Order("scope_type ASC, scope_id ASC").Find(&rules).Error
	return rules, exception.HandleGormError(err)
}

func (r *ruleRepository) DeleteByScope(ctx context.Context, scopeType string, scopeID uint) error {
	res := r.db.WithContext(ctx).Where("scope_type = ? AND scope_id = ?", scopeType, scopeID).Delete(&domain.BookingRule{})
	if res.Error != nil {
		return exception.HandleGormError(res.Error)
	}
	if res.RowsAffected == 0 {
		return exception.HandleGormError(gorm.ErrRecordNotFound)
	}
	return nil
}
