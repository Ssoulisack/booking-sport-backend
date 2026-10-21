package postgres

import (
	"backend/internal/adapters/secondary/postgres/exception"
	"backend/internal/core/domain"
	"backend/internal/core/port/output"
	"context"
	"errors"

	"gorm.io/gorm"
)

// requireAffected maps "no row changed" to not found
func requireAffected(res *gorm.DB) error {
	if res.Error != nil {
		return exception.HandleGormError(res.Error)
	}
	if res.RowsAffected == 0 {
		return exception.HandleGormError(gorm.ErrRecordNotFound)
	}
	return nil
}

type openingHourRepository struct{ db *gorm.DB }

func NewOpeningHourRepository(db *gorm.DB) output.OpeningHourRepository {
	return &openingHourRepository{db: db}
}

func (r *openingHourRepository) ListByOwner(ctx context.Context, businessID uint, ownerType string, ownerID uint) ([]domain.OpeningHour, error) {
	if businessID == 0 {
		return nil, errNoScope
	}
	var hours []domain.OpeningHour
	err := r.db.WithContext(ctx).
		Where("business_id = ? AND owner_type = ? AND owner_id = ?", businessID, ownerType, ownerID).
		Order("weekday ASC, open_time ASC").Find(&hours).Error
	return hours, exception.HandleGormError(err)
}

// ReplaceForOwner hard-deletes the old schedule: hours have no history worth keeping
func (r *openingHourRepository) ReplaceForOwner(ctx context.Context, businessID uint, ownerType string, ownerID uint, hours []domain.OpeningHour) error {
	if businessID == 0 {
		return errNoScope
	}
	return exception.HandleGormError(r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Unscoped().
			Where("business_id = ? AND owner_type = ? AND owner_id = ?", businessID, ownerType, ownerID).
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

func (r *priceRepository) GetByOwner(ctx context.Context, businessID uint, ownerType string, ownerID uint) (*domain.Price, error) {
	if businessID == 0 {
		return nil, errNoScope
	}
	var p domain.Price
	err := r.db.WithContext(ctx).
		Where("business_id = ? AND owner_type = ? AND owner_id = ?", businessID, ownerType, ownerID).First(&p).Error
	if err != nil {
		return nil, exception.HandleGormError(err)
	}
	return &p, nil
}

func (r *priceRepository) Upsert(ctx context.Context, p *domain.Price) error {
	if p.BusinessID == 0 {
		return errNoScope
	}
	return exception.HandleGormError(r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing domain.Price
		err := tx.Where("business_id = ? AND owner_type = ? AND owner_id = ?", p.BusinessID, p.OwnerType, p.OwnerID).
			First(&existing).Error
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

func (r *priceRepository) DeleteByOwner(ctx context.Context, businessID uint, ownerType string, ownerID uint) error {
	if businessID == 0 {
		return errNoScope
	}
	return requireAffected(r.db.WithContext(ctx).
		Where("business_id = ? AND owner_type = ? AND owner_id = ?", businessID, ownerType, ownerID).
		Delete(&domain.Price{}))
}

type ruleRepository struct{ db *gorm.DB }

func NewRuleRepository(db *gorm.DB) output.RuleRepository {
	return &ruleRepository{db: db}
}

func (r *ruleRepository) GetByScope(ctx context.Context, businessID uint, scopeType string, scopeID uint) (*domain.BookingRule, error) {
	if businessID == 0 {
		return nil, errNoScope
	}
	var rule domain.BookingRule
	err := r.db.WithContext(ctx).
		Where("business_id = ? AND scope_type = ? AND scope_id = ?", businessID, scopeType, scopeID).First(&rule).Error
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
	if rule.BusinessID == 0 {
		return errNoScope
	}
	return exception.HandleGormError(r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing domain.BookingRule
		err := tx.Where("business_id = ? AND scope_type = ? AND scope_id = ?", rule.BusinessID, rule.ScopeType, rule.ScopeID).
			First(&existing).Error
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

func (r *ruleRepository) List(ctx context.Context, businessID uint) ([]domain.BookingRule, error) {
	if businessID == 0 {
		return nil, errNoScope
	}
	var rules []domain.BookingRule
	err := r.db.WithContext(ctx).Where("business_id = ?", businessID).
		Order("scope_type ASC, scope_id ASC").Find(&rules).Error
	return rules, exception.HandleGormError(err)
}

func (r *ruleRepository) DeleteByScope(ctx context.Context, businessID uint, scopeType string, scopeID uint) error {
	if businessID == 0 {
		return errNoScope
	}
	return requireAffected(r.db.WithContext(ctx).
		Where("business_id = ? AND scope_type = ? AND scope_id = ?", businessID, scopeType, scopeID).
		Delete(&domain.BookingRule{}))
}

type mediaRepository struct{ db *gorm.DB }

func NewMediaRepository(db *gorm.DB) output.MediaRepository {
	return &mediaRepository{db: db}
}

func (r *mediaRepository) Create(ctx context.Context, m *domain.Media) error {
	if m.BusinessID == 0 {
		return errNoScope
	}
	return exception.HandleGormError(r.db.WithContext(ctx).Create(m).Error)
}

func (r *mediaRepository) Get(ctx context.Context, businessID uint, key string) (*domain.Media, error) {
	if businessID == 0 {
		return nil, errNoScope
	}
	var m domain.Media
	err := r.db.WithContext(ctx).Where("business_id = ? AND key = ?", businessID, key).First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, exception.HandleGormError(err)
	}
	return &m, nil
}

func (r *mediaRepository) Delete(ctx context.Context, businessID uint, key string) error {
	if businessID == 0 {
		return errNoScope
	}
	return requireAffected(r.db.WithContext(ctx).Where("business_id = ? AND key = ?", businessID, key).Delete(&domain.Media{}))
}

func (r *mediaRepository) Usage(ctx context.Context, businessID uint) (int64, int64, error) {
	if businessID == 0 {
		return 0, 0, errNoScope
	}
	var row struct {
		Images int64
		Bytes  int64
	}
	err := r.db.WithContext(ctx).Model(&domain.Media{}).Where("business_id = ?", businessID).
		Select("COUNT(*) AS images, COALESCE(SUM(size_bytes), 0) AS bytes").Scan(&row).Error
	return row.Images, row.Bytes, exception.HandleGormError(err)
}
