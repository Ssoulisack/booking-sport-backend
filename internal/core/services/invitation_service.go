package services

import (
	"backend/internal/core/domain"
	"backend/internal/core/dto"
	"backend/internal/core/port/input"
	"backend/internal/core/port/output"
	"backend/pkg/response"
	"context"
	"log"
	"net/http"
	"net/mail"
	"strings"
	"time"
)

const inviteTTL = 7 * 24 * time.Hour

// invites creates and re-sends invitations; shared by the platform (owner invites)
// and by business owners (manager / staff invites)
type invites struct {
	invitations output.InvitationRepository
	memberships output.MembershipRepository
	users       output.UserRepository
	limits      *Limits
	mailer      output.Mailer
	frontendURL string
	now         func() time.Time
}

func newInvites(
	invitations output.InvitationRepository,
	memberships output.MembershipRepository,
	users output.UserRepository,
	limits *Limits,
	mailer output.Mailer,
	frontendURL string,
) invites {
	return invites{
		invitations: invitations, memberships: memberships, users: users, limits: limits,
		mailer: mailer, frontendURL: strings.TrimRight(frontendURL, "/"), now: time.Now,
	}
}

func toInvitationDTO(inv domain.Invitation) dto.InvitationDTO {
	return dto.InvitationDTO{ID: inv.ID, Email: inv.Email, Role: inv.Role, ExpiresAt: inv.ExpiresAt}
}

func (iv invites) acceptURL(raw string) string { return iv.frontendURL + "/invite/" + raw }

// create stores an invitation for email and emails the link. If the email cannot be sent the
// invitation is removed again, so a seat is not held by an invite nobody received.
func (iv invites) create(ctx context.Context, b *domain.Business, actor *domain.User, email, role string) (*dto.InvitationDTO, error) {
	email = normalizeEmail(email)
	if addr, err := mail.ParseAddress(email); err != nil || addr.Address != email {
		return nil, badRequest("INVALID_EMAIL")
	}
	if existing, err := iv.users.FindByEmail(ctx, email); err == nil {
		m, err := iv.memberships.Get(ctx, b.ID, existing.ID)
		if err != nil {
			return nil, err
		}
		if m != nil {
			return nil, conflict("ALREADY_A_MEMBER")
		}
	} else if !isNotFound(err) {
		return nil, err
	}
	pending, err := iv.invitations.PendingForEmail(ctx, b.ID, email, iv.now())
	if err != nil {
		return nil, err
	}
	if pending != nil {
		return nil, conflict("INVITATION_ALREADY_PENDING")
	}
	if err := iv.limits.CheckMembers(ctx, b.ID); err != nil {
		return nil, err
	}

	raw, hash, err := newToken()
	if err != nil {
		return nil, err
	}
	inv := domain.Invitation{
		BusinessID: b.ID, Email: email, Role: role, TokenHash: hash,
		ExpiresAt: iv.now().Add(inviteTTL), InvitedByID: actor.ID,
	}
	if err := iv.invitations.Create(ctx, &inv); err != nil {
		return nil, err
	}
	if err := iv.mailer.SendInvitation(email, b.Name, actor.Name, role, iv.acceptURL(raw)); err != nil {
		log.Printf("❌ invitation email to %s failed: %v", email, err)
		_ = iv.invitations.Delete(ctx, b.ID, inv.ID)
		return nil, response.NewError(http.StatusBadGateway, "EMAIL_SEND_FAILED")
	}
	d := toInvitationDTO(inv)
	return &d, nil
}

// resend issues a fresh link (the old one stops working) and a new 7-day expiry
func (iv invites) resend(ctx context.Context, b *domain.Business, actor *domain.User, inv *domain.Invitation) (*dto.InvitationDTO, error) {
	if inv.AcceptedAt != nil {
		return nil, conflict("INVITATION_ALREADY_ACCEPTED")
	}
	raw, hash, err := newToken()
	if err != nil {
		return nil, err
	}
	inv.TokenHash, inv.ExpiresAt = hash, iv.now().Add(inviteTTL)
	if err := iv.invitations.Update(ctx, inv); err != nil {
		return nil, err
	}
	if err := iv.mailer.SendInvitation(inv.Email, b.Name, actor.Name, inv.Role, iv.acceptURL(raw)); err != nil {
		log.Printf("❌ invitation email to %s failed: %v", inv.Email, err)
		return nil, response.NewError(http.StatusBadGateway, "EMAIL_SEND_FAILED")
	}
	d := toInvitationDTO(*inv)
	return &d, nil
}

// ---------- accepting ----------

type InvitationServiceImpl struct {
	invitations output.InvitationRepository
	memberships output.MembershipRepository
	businesses  output.BusinessRepository
	users       output.UserRepository
	now         func() time.Time
}

func NewInvitationService(
	invitations output.InvitationRepository,
	memberships output.MembershipRepository,
	businesses output.BusinessRepository,
	users output.UserRepository,
) input.InvitationService {
	return &InvitationServiceImpl{
		invitations: invitations, memberships: memberships, businesses: businesses, users: users, now: time.Now,
	}
}

// open finds a usable invitation and its business; used, expired and unknown links look the same
func (s *InvitationServiceImpl) open(ctx context.Context, token string) (*domain.Invitation, *domain.Business, error) {
	gone := notFound("INVITATION_NOT_FOUND_OR_EXPIRED")
	inv, err := s.invitations.GetByTokenHash(ctx, hashToken(token))
	if err != nil {
		return nil, nil, err
	}
	if inv == nil || inv.AcceptedAt != nil || !s.now().Before(inv.ExpiresAt) {
		return nil, nil, gone
	}
	b, err := s.businesses.GetByID(ctx, inv.BusinessID)
	if err != nil {
		if isNotFound(err) {
			return nil, nil, gone
		}
		return nil, nil, err
	}
	return inv, b, nil
}

func (s *InvitationServiceImpl) GetInvitation(ctx context.Context, token string) (*dto.InvitationInfoDTO, error) {
	inv, b, err := s.open(ctx, token)
	if err != nil {
		return nil, err
	}
	exists, err := s.users.ExistsByEmail(ctx, inv.Email)
	if err != nil {
		return nil, err
	}
	return &dto.InvitationInfoDTO{BusinessName: b.Name, Email: inv.Email, Role: inv.Role, AccountExists: exists}, nil
}

func (s *InvitationServiceImpl) Accept(ctx context.Context, token string, actor *domain.User, req dto.AcceptInvitationRequest) (*dto.AuthResponseDTO, error) {
	inv, b, err := s.open(ctx, token)
	if err != nil {
		return nil, err
	}
	if b.Suspended {
		return nil, forbidden("BUSINESS_SUSPENDED")
	}

	user, err := s.users.FindByEmail(ctx, inv.Email)
	switch {
	case err == nil:
		// an existing account must prove it is the invited person by being logged in as them
		if actor == nil || actor.ID != user.ID {
			return nil, response.NewError(http.StatusUnauthorized, "LOGIN_AS_INVITED_EMAIL")
		}
	case isNotFound(err):
		user, err = createUser(ctx, s.users, dto.RegisterDTO{Name: req.Name, Email: inv.Email, Password: req.Password, Phone: req.Phone})
		if err != nil {
			return nil, err
		}
	default:
		return nil, err
	}

	existing, err := s.memberships.Get(ctx, b.ID, user.ID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, conflict("ALREADY_A_MEMBER")
	}
	if inv.Role == domain.MemberOwner {
		hasOwner, err := s.memberships.HasOwner(ctx, b.ID)
		if err != nil {
			return nil, err
		}
		if hasOwner {
			return nil, conflict("BUSINESS_ALREADY_HAS_OWNER")
		}
	}
	if err := s.memberships.Create(ctx, &domain.Membership{UserID: user.ID, BusinessID: b.ID, Role: inv.Role, Active: true}); err != nil {
		return nil, err
	}
	now := s.now()
	inv.AcceptedAt = &now
	if err := s.invitations.Update(ctx, inv); err != nil {
		return nil, err
	}
	return issueTokens(user)
}
