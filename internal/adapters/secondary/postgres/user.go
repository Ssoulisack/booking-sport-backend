package postgres

import (
	"backend/internal/adapters/secondary/postgres/exception"
	"backend/internal/core/domain"
	"backend/internal/core/port/output"
	"context"

	"gorm.io/gorm"
)

// userRepository implements the UserRepository interface using PostgreSQL
type userRepository struct {
	db *gorm.DB
}

// NewUserRepository creates a new PostgreSQL user repository
func NewUserRepository(db *gorm.DB) output.UserRepository {
	return &userRepository{db: db}
}

var _ output.UserRepository = (*userRepository)(nil)

func (r *userRepository) Create(ctx context.Context, user *domain.User) error {
	return exception.HandleGormError(r.db.WithContext(ctx).Create(user).Error)
}

func (r *userRepository) FindByID(ctx context.Context, id uint) (*domain.User, error) {
	var user domain.User
	if err := r.db.WithContext(ctx).First(&user, id).Error; err != nil {
		return nil, exception.HandleGormError(err)
	}
	return &user, nil
}

func (r *userRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	var user domain.User
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error; err != nil {
		return nil, exception.HandleGormError(err)
	}
	return &user, nil
}

func (r *userRepository) List(ctx context.Context, q output.ListQuery) ([]domain.User, int64, error) {
	tx := r.db.WithContext(ctx).Model(&domain.User{})
	if q.Keyword != "" {
		like := "%" + q.Keyword + "%"
		tx = tx.Where("name ILIKE ? OR email ILIKE ?", like, like)
	}
	for column, value := range q.Filters {
		tx = tx.Where(column+" = ?", value)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, exception.HandleGormError(err)
	}
	var users []domain.User
	tx = tx.Order("id ASC")
	if q.Limit > 0 {
		tx = tx.Offset((max(q.Page, 1) - 1) * q.Limit).Limit(q.Limit)
	}
	if err := tx.Find(&users).Error; err != nil {
		return nil, 0, exception.HandleGormError(err)
	}
	return users, total, nil
}

func (r *userRepository) SetStatus(ctx context.Context, id uint, status bool) error {
	res := r.db.WithContext(ctx).Model(&domain.User{}).Where("id = ?", id).Update("status", status)
	if res.Error != nil {
		return exception.HandleGormError(res.Error)
	}
	if res.RowsAffected == 0 {
		return exception.HandleGormError(gorm.ErrRecordNotFound)
	}
	return nil
}

func (r *userRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&domain.User{}).Where("email = ?", email).Count(&count).Error
	return count > 0, exception.HandleGormError(err)
}
