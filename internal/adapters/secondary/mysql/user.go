package mysql

import (
	"backend/internal/core/port/output"

	"gorm.io/gorm"
)

// userRepository implements the UserRepository interface using PostgreSQL
type UserRepositoryImpl struct {
	db *gorm.DB
}

// NewUserRepository creates a new PostgreSQL user repository
func NewUserRepository(db *gorm.DB) output.UserRepository {
	return &UserRepositoryImpl{
		db: db,
	}
}

var _ output.UserRepository = (*UserRepositoryImpl)(nil)
