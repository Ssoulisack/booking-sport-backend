package services

import (
	"backend/internal/core/domain"
	"backend/internal/core/dto"
	"backend/internal/core/port/input"
	"backend/internal/core/port/output"
	"context"
)

type AdminUserServiceImpl struct {
	userRepo output.UserRepository
}

func NewAdminUserService(userRepo output.UserRepository) input.AdminUserService {
	return &AdminUserServiceImpl{userRepo: userRepo}
}

// CreateStaff always creates a staff account; admin accounts are not created through the API
func (s *AdminUserServiceImpl) CreateStaff(ctx context.Context, req dto.CreateStaffRequest) (*dto.UserDTO, error) {
	user, err := createUser(ctx, s.userRepo, dto.RegisterDTO(req), domain.RoleStaff)
	if err != nil {
		return nil, err
	}
	d := toUserDTO(user)
	return &d, nil
}

func (s *AdminUserServiceImpl) ListUsers(ctx context.Context, q dto.ListQuery) (*dto.PagedResult[dto.UserDTO], error) {
	lq, page, limit := newListQuery(q)
	if q.RoleID != 0 {
		lq.Filters["role_id"] = q.RoleID
	}
	users, total, err := s.userRepo.List(ctx, lq)
	if err != nil {
		return nil, err
	}
	return pageOf(users, total, page, limit, func(u domain.User) dto.UserDTO { return toUserDTO(&u) }), nil
}

// SetUserStatus enables or disables an account. An admin cannot disable themselves.
// A disabled user cannot log in or refresh, but an access token already issued stays valid until it expires.
func (s *AdminUserServiceImpl) SetUserStatus(ctx context.Context, actorID, targetID uint, status bool) error {
	if actorID == targetID && !status {
		return badRequest("CANNOT_DISABLE_YOURSELF")
	}
	return s.userRepo.SetStatus(ctx, targetID, status)
}
