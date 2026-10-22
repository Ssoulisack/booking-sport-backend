package postgres

import (
	"backend/internal/adapters/secondary/postgres/exception"
	"backend/internal/core/domain"
	"backend/internal/core/port/output"
	"context"

	"gorm.io/gorm"
)

type directoryRepository struct{ db *gorm.DB }

func NewDirectoryRepository(db *gorm.DB) output.DirectoryRepository {
	return &directoryRepository{db: db}
}

func (r *directoryRepository) Businesses(ctx context.Context, q output.DirectoryQuery) ([]domain.Business, int64, error) {
	tx := r.db.WithContext(ctx).Model(&domain.Business{}).Where("published AND NOT suspended")
	if q.Keyword != "" {
		like := "%" + q.Keyword + "%"
		tx = tx.Where("(name ILIKE ? OR address ILIKE ?)", like, like)
	}
	if q.Province != "" {
		tx = tx.Where("province = ?", q.Province)
	}
	if q.SportType != "" {
		tx = tx.Where(`EXISTS (SELECT 1 FROM sports s WHERE s.business_id = businesses.id
			AND s.active AND s.deleted_at IS NULL AND s.type = ?)`, q.SportType)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, exception.HandleGormError(err)
	}
	var items []domain.Business
	err := paginate(tx.Order("published_at DESC NULLS LAST, id DESC"), output.ListQuery{Page: q.Page, Limit: q.Limit}).Find(&items).Error
	return items, total, exception.HandleGormError(err)
}

func (r *directoryRepository) ActiveSports(ctx context.Context, businessIDs []uint) (map[uint][]domain.Sport, error) {
	out := map[uint][]domain.Sport{}
	if len(businessIDs) == 0 {
		return out, nil
	}
	var sports []domain.Sport
	err := r.db.WithContext(ctx).Where("business_id IN ? AND active", businessIDs).Order("name ASC").Find(&sports).Error
	if err != nil {
		return nil, exception.HandleGormError(err)
	}
	for _, s := range sports {
		out[s.BusinessID] = append(out[s.BusinessID], s)
	}
	return out, nil
}

func (r *directoryRepository) LowestCourtPrices(ctx context.Context, businessIDs []uint) (map[uint]domain.Price, error) {
	out := map[uint]domain.Price{}
	if len(businessIDs) == 0 {
		return out, nil
	}
	var rows []domain.Price
	err := r.db.WithContext(ctx).
		Table("prices p").
		Select("DISTINCT ON (p.business_id) p.business_id, p.price_per_hour, p.currency").
		Joins("JOIN resources r ON r.id = p.owner_id AND r.deleted_at IS NULL AND r.active").
		Where("p.owner_type = ? AND p.deleted_at IS NULL AND p.business_id IN ?", domain.OwnerResource, businessIDs).
		Order("p.business_id, p.price_per_hour ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, exception.HandleGormError(err)
	}
	for _, p := range rows {
		out[p.BusinessID] = p
	}
	return out, nil
}
