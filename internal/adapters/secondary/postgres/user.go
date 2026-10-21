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

func (r *userRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&domain.User{}).Where("email = ?", email).Count(&count).Error
	return count > 0, exception.HandleGormError(err)
}
