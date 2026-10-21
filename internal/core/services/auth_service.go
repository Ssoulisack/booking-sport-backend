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
	"log"
	"net/http"
	"net/mail"
	"strings"
	"time"
)

const (
	minPasswordLength = 8
	resetTokenTTL     = time.Hour
)

type AuthServiceImpl struct {
	userRepo    output.UserRepository
	cacheRepo   redis.CacheRepository
	memberships output.MembershipRepository
	resets      output.PasswordResetRepository
	mailer      output.Mailer
	frontendURL string
	now         func() time.Time
}

func NewAuthService(
	userRepo output.UserRepository,
	cacheRepo redis.CacheRepository,
	memberships output.MembershipRepository,
	resets output.PasswordResetRepository,
	mailer output.Mailer,
	frontendURL string,
) input.AuthService {
	return &AuthServiceImpl{
		userRepo:    userRepo,
		cacheRepo:   cacheRepo,
		memberships: memberships,
		resets:      resets,
		mailer:      mailer,
		frontendURL: strings.TrimRight(frontendURL, "/"),
		now:         time.Now,
	}
}

// Register creates a customer account. Business members join through invitations.
func (s *AuthServiceImpl) Register(ctx context.Context, req dto.RegisterDTO) (*dto.AuthResponseDTO, error) {
	user, err := createUser(ctx, s.userRepo, req)
	if err != nil {
		return nil, err
	}
	return issueTokens(user)
}

func validPassword(password string) error {
	if len(password) < minPasswordLength {
		return badRequest("PASSWORD_TOO_SHORT")
	}
	return nil
}

// createUser validates a registration and stores a plain account (no platform role)
func createUser(ctx context.Context, repo output.UserRepository, req dto.RegisterDTO) (*domain.User, error) {
	email := normalizeEmail(req.Email)
	if addr, err := mail.ParseAddress(email); err != nil || addr.Address != email {
		return nil, badRequest("INVALID_EMAIL")
	}
	if strings.TrimSpace(req.Name) == "" {
		return nil, badRequest("NAME_IS_REQUIRED")
	}
	if err := validPassword(req.Password); err != nil {
		return nil, err
	}

	exists, err := repo.ExistsByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, conflict("EMAIL_ALREADY_REGISTERED")
	}

	hash, err := utils.Encrypt(req.Password)
	if err != nil {
		return nil, err
	}
	user := &domain.User{
		Email:        email,
		Username:     email, // username is unique but accounts are identified by email only
		Name:         strings.TrimSpace(req.Name),
		Phone:        strings.TrimSpace(req.Phone),
		PasswordHash: hash,
		Status:       true,
		LastActive:   time.Now(),
	}
	if err := repo.Create(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
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
		return nil, forbidden("ACCOUNT_DISABLED")
	}
	return issueTokens(user)
}

// checkUsable refuses disabled accounts and tokens issued before the last password reset
func checkUsable(u *domain.User, issuedAt int64) error {
	if !u.Status {
		return forbidden("ACCOUNT_DISABLED")
	}
	if u.PasswordChangedAt != nil && issuedAt < u.PasswordChangedAt.Unix() {
		return response.NewError(http.StatusUnauthorized, "TOKEN_REVOKED")
	}
	return nil
}

// Refresh re-reads the user so a disabled account or a password reset takes effect
func (s *AuthServiceImpl) Refresh(ctx context.Context, userID uint, issuedAt int64) (*dto.AuthResponseDTO, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if err := checkUsable(user, issuedAt); err != nil {
		return nil, err
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

func (s *AuthServiceImpl) MyBusinesses(ctx context.Context, userID uint) ([]dto.MyBusinessDTO, error) {
	memberships, err := s.memberships.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	return mapSlice(memberships, func(m domain.Membership) dto.MyBusinessDTO {
		return dto.MyBusinessDTO{
			BusinessID: m.BusinessID, Name: m.Business.Name, Slug: m.Business.Slug,
			Status: m.Business.Status(), Role: m.Role, Active: m.Active,
		}
	}), nil
}

// ForgotPassword emails a one-hour reset link. Unknown or disabled accounts get no email,
// but the caller sees the same answer either way.
func (s *AuthServiceImpl) ForgotPassword(ctx context.Context, email string) error {
	user, err := s.userRepo.FindByEmail(ctx, normalizeEmail(email))
	if err != nil {
		if isNotFound(err) {
			return nil
		}
		return err
	}
	if !user.Status {
		return nil
	}
	raw, hash, err := newToken()
	if err != nil {
		return err
	}
	reset := domain.PasswordReset{UserID: user.ID, TokenHash: hash, ExpiresAt: s.now().Add(resetTokenTTL)}
	if err := s.resets.Create(ctx, &reset); err != nil {
		return err
	}
	if err := s.mailer.SendPasswordReset(user.Email, user.Name, s.frontendURL+"/reset-password/"+raw); err != nil {
		log.Printf("❌ password reset email to user %d failed: %v", user.ID, err)
	}
	return nil
}

// ResetPassword sets a new password from a valid link; every older token stops working
func (s *AuthServiceImpl) ResetPassword(ctx context.Context, req dto.ResetPasswordRequest) error {
	if err := validPassword(req.Password); err != nil {
		return err
	}
	reset, err := s.resets.GetByTokenHash(ctx, hashToken(req.Token))
	if err != nil {
		return err
	}
	now := s.now()
	if reset == nil || reset.UsedAt != nil || now.After(reset.ExpiresAt) {
		return badRequest("INVALID_OR_EXPIRED_TOKEN")
	}
	hash, err := utils.Encrypt(req.Password)
	if err != nil {
		return err
	}
	if err := s.userRepo.SetPassword(ctx, reset.UserID, hash, now); err != nil {
		return err
	}
	return s.resets.UseAll(ctx, reset.UserID, now)
}

func issueTokens(user *domain.User) (*dto.AuthResponseDTO, error) {
	// role claim unused: roles are read from the database on every request
	pair, err := jwt.GenerateJWTToken(int(user.ID), 0)
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
		ID:           u.ID,
		Email:        u.Email,
		Avatar:       u.Avatar,
		Name:         u.Name,
		Username:     u.Username,
		Phone:        u.Phone,
		PlatformRole: u.PlatformRole,
		Status:       u.Status,
		CreatedAt:    u.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    u.UpdatedAt.Format(time.RFC3339),
	}
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func isNotFound(err error) bool {
	var appErr response.AppError
	return errors.As(err, &appErr) && appErr.Status == http.StatusNotFound
}
