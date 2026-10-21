package output

import (
	"backend/internal/core/domain"
	"context"
)

type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	FindByID(ctx context.Context, id uint) (*domain.User, error)
	FindByEmail(ctx context.Context, email string) (*domain.User, error)
	ExistsByEmail(ctx context.Context, email string) (bool, error)
	// List supports the filter "role_id" and a keyword matched against name and email
	List(ctx context.Context, q ListQuery) ([]domain.User, int64, error)
	SetStatus(ctx context.Context, id uint, status bool) error
}
