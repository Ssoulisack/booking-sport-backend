package services

import (
	"backend/internal/adapters/secondary/redis"
	"backend/internal/core/domain"
	"backend/internal/core/dto"
	"backend/internal/core/port/input"
	"backend/internal/core/port/output"
	"backend/pkg/jwt"
	"backend/pkg/response"
	"backend/pkg/utils"
	"context"
	"errors"
	"net/http"
	"net/mail"
	"strings"
	"time"
)

const minPasswordLength = 8

type AuthServiceImpl struct {
	userRepo  output.UserRepository
	cacheRepo redis.CacheRepository
}

func NewAuthService(
	userRepo output.UserRepository,
	cacheRepo redis.CacheRepository,
) input.AuthService {
	return &AuthServiceImpl{
		userRepo:  userRepo,
		cacheRepo: cacheRepo,
	}
}

// Register creates a client account; staff and admin accounts are created by an admin
func (s *AuthServiceImpl) Register(ctx context.Context, req dto.RegisterDTO) (*dto.AuthResponseDTO, error) {
	email := normalizeEmail(req.Email)
	if addr, err := mail.ParseAddress(email); err != nil || addr.Address != email {
		return nil, response.ErrorBadRequest("INVALID_EMAIL")
	}
	if strings.TrimSpace(req.Name) == "" {
		return nil, response.ErrorBadRequest("NAME_IS_REQUIRED")
	}
	if len(req.Password) < minPasswordLength {
		return nil, response.ErrorBadRequest("PASSWORD_TOO_SHORT")
	}

	exists, err := s.userRepo.ExistsByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, response.NewError(http.StatusConflict, "EMAIL_ALREADY_REGISTERED")
	}

	hash, err := utils.Encrypt(req.Password)
	if err != nil {
		return nil, err
	}
	user := &domain.User{
		Email:        email,
		Username:     email, // username is unique but clients register by email only
		Name:         strings.TrimSpace(req.Name),
		Phone:        strings.TrimSpace(req.Phone),
		PasswordHash: hash,
		RoleID:       domain.RoleClient,
		Status:       true,
		LastActive:   time.Now(),
	}
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}
	return issueTokens(user)
}

func (s *AuthServiceImpl) Login(ctx context.Context, req dto.LoginDTO) (*dto.AuthResponseDTO, error) {
	invalid := response.NewError(http.StatusUnauthorized, "INVALID_CREDENTIALS")

	user, err := s.userRepo.FindByEmail(ctx, normalizeEmail(req.Email))
	if err != nil {
		if isNotFound(err) {
			return nil, invalid
		}
		return nil, err
	}
	if utils.VerifyPassword(string(user.PasswordHash), req.Password) != nil {
		return nil, invalid
	}
	if !user.Status {
		return nil, response.NewError(http.StatusForbidden, "ACCOUNT_DISABLED")
	}
	return issueTokens(user)
}

// Refresh re-reads the user so a changed role or a disabled account takes effect
func (s *AuthServiceImpl) Refresh(ctx context.Context, userID uint) (*dto.AuthResponseDTO, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !user.Status {
		return nil, response.NewError(http.StatusForbidden, "ACCOUNT_DISABLED")
	}
	return issueTokens(user)
}

func (s *AuthServiceImpl) Me(ctx context.Context, userID uint) (*dto.UserDTO, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	d := toUserDTO(user)
	return &d, nil
}

func issueTokens(user *domain.User) (*dto.AuthResponseDTO, error) {
	pair, err := jwt.GenerateJWTToken(int(user.ID), int(user.RoleID))
	if err != nil {
		return nil, err
	}
	return &dto.AuthResponseDTO{
		// TokenPair holds the tokens as quoted JSON strings
		AccessToken:  strings.Trim(string(pair.AccessToken), `"`),
		RefreshToken: strings.Trim(string(pair.RefreshToken), `"`),
		User:         toUserDTO(user),
	}, nil
}

func toUserDTO(u *domain.User) dto.UserDTO {
	return dto.UserDTO{
		ID:        u.ID,
		Email:     u.Email,
		Avatar:    u.Avatar,
		Name:      u.Name,
		Username:  u.Username,
		Phone:     u.Phone,
		RoleID:    u.RoleID,
		Status:    u.Status,
		CreatedAt: u.CreatedAt.Format(time.RFC3339),
		UpdatedAt: u.UpdatedAt.Format(time.RFC3339),
	}
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func isNotFound(err error) bool {
	var appErr response.AppError
	return errors.As(err, &appErr) && appErr.Status == http.StatusNotFound
}
