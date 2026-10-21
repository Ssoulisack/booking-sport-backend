package postgres

import (
	"backend/internal/adapters/secondary/postgres/exception"
	"backend/internal/core/port/output"
	"context"

	"gorm.io/gorm"
)

// crudRepository is a generic gorm implementation of output.CRUDRepository.
// searchColumn is the column matched (ILIKE) by ListQuery.Keyword; empty disables keyword search.
type crudRepository[T any] struct {
	db           *gorm.DB
	searchColumn string
	preloads     []string
}

// NewCRUDRepository creates a PostgreSQL repository for any gorm model
func NewCRUDRepository[T any](db *gorm.DB, searchColumn string, preloads ...string) output.CRUDRepository[T] {
	return newCRUD[T](db, searchColumn, preloads...)
}

func newCRUD[T any](db *gorm.DB, searchColumn string, preloads ...string) *crudRepository[T] {
	return &crudRepository[T]{db: db, searchColumn: searchColumn, preloads: preloads}
}

func (r *crudRepository[T]) withPreloads(tx *gorm.DB) *gorm.DB {
	for _, p := range r.preloads {
		tx = tx.Preload(p)
	}
	return tx
}

func (r *crudRepository[T]) Create(ctx context.Context, m *T) error {
	return exception.HandleGormError(r.db.WithContext(ctx).Create(m).Error)
}

func (r *crudRepository[T]) GetByID(ctx context.Context, id uint) (*T, error) {
	var m T
	if err := r.withPreloads(r.db.WithContext(ctx)).First(&m, id).Error; err != nil {
		return nil, exception.HandleGormError(err)
	}
	return &m, nil
}

func (r *crudRepository[T]) List(ctx context.Context, q output.ListQuery) ([]T, int64, error) {
	tx := r.db.WithContext(ctx).Model(new(T))
	if q.Keyword != "" && r.searchColumn != "" {
		tx = tx.Where(r.searchColumn+" ILIKE ?", "%"+q.Keyword+"%")
	}
	for column, value := range q.Filters {
		tx = tx.Where(column+" = ?", value)
	}

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, exception.HandleGormError(err)
	}

	var items []T
	tx = r.withPreloads(tx).Order("id ASC")
	if q.Limit > 0 {
		tx = tx.Offset((max(q.Page, 1) - 1) * q.Limit).Limit(q.Limit)
	}
	if err := tx.Find(&items).Error; err != nil {
		return nil, 0, exception.HandleGormError(err)
	}
	return items, total, nil
}

func (r *crudRepository[T]) Update(ctx context.Context, m *T) error {
	return exception.HandleGormError(r.db.WithContext(ctx).Save(m).Error)
}

func (r *crudRepository[T]) Delete(ctx context.Context, id uint) error {
	res := r.db.WithContext(ctx).Delete(new(T), id)
	if res.Error != nil {
		return exception.HandleGormError(res.Error)
	}
	if res.RowsAffected == 0 {
		return exception.HandleGormError(gorm.ErrRecordNotFound)
	}
	return nil
}
