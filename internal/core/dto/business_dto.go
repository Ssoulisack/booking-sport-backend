package dto

import (
	"backend/internal/core/domain"
	"time"
)

// ---------- platform (Super Admin) ----------

type PlanDTO struct {
	ID     uint              `json:"id"`
	Name   string            `json:"name"`
	Limits domain.PlanLimits `json:"limits"`
}

type PlanRequest struct {
	Limits domain.PlanLimits `json:"limits"`
}

type CreateBusinessRequest struct {
	Name   string `json:"name"`
	Slug   string `json:"slug"` // optional; generated from the name when empty
	PlanID uint   `json:"plan_id"`
}

type UpdateBusinessRequest struct {
	Name      string                `json:"name"`
	PlanID    uint                  `json:"plan_id"`
	Overrides domain.LimitOverrides `json:"overrides"`
}

type BusinessDTO struct {
	ID          uint                  `json:"id"`
	Name        string                `json:"name"`
	Slug        string                `json:"slug"`
	Status      string                `json:"status"`
	Plan        PlanDTO               `json:"plan"`
	Overrides   domain.LimitOverrides `json:"overrides"`
	Limits      domain.PlanLimits     `json:"limits"` // plan merged with overrides
	PublishedAt *time.Time            `json:"published_at"`
	CreatedAt   time.Time             `json:"created_at"`
}

type InviteOwnerRequest struct {
	Email string `json:"email"`
}

// ---------- business settings ----------

type SettingsRequest struct {
	Name            string            `json:"name"`
	Slug            string            `json:"slug"` // can change only until the first publish
	Timezone        string            `json:"timezone"`
	Currency        string            `json:"currency"`
	Language        string            `json:"language"`
	LogoKey         string            `json:"logo_key"`
	CoverKey        string            `json:"cover_key"`
	BrandColor      string            `json:"brand_color"`
	Description     string            `json:"description"`
	Address         string            `json:"address"`
	Phone           string            `json:"phone"`
	MapURL          string            `json:"map_url"`
	SocialLinks     map[string]string `json:"social_links"`
	BookingPolicy   string            `json:"booking_policy"`
	BookingMode     string            `json:"booking_mode"`
	RequireApproval bool              `json:"require_approval"`
}

type SettingsDTO struct {
	SettingsRequest
	ID       uint   `json:"id"`
	Status   string `json:"status"`
	LogoURL  string `json:"logo_url"`
	CoverURL string `json:"cover_url"`
	// PublicPath is the shareable booking page path, e.g. /b/smash-tennis
	PublicPath  string     `json:"public_path"`
	PublishedAt *time.Time `json:"published_at"`
}

type UsageItem struct {
	Used  int64 `json:"used"`
	Limit int64 `json:"limit"`
}

type UsageDTO struct {
	Plan                   string    `json:"plan"`
	Sports                 UsageItem `json:"sports"`
	Resources              UsageItem `json:"resources"`
	Coaches                UsageItem `json:"coaches"`
	ClassSessionsThisMonth UsageItem `json:"class_sessions_this_month"`
	Members                UsageItem `json:"members"` // members + pending invites
	StorageMB              UsageItem `json:"storage_mb"`
	Images                 UsageItem `json:"images"`
}

// ---------- members and invitations ----------

type MemberDTO struct {
	UserID   uint      `json:"user_id"`
	Name     string    `json:"name"`
	Email    string    `json:"email"`
	Role     string    `json:"role"`
	Active   bool      `json:"active"`
	JoinedAt time.Time `json:"joined_at"`
}

type UpdateMemberRequest struct {
	Role   *string `json:"role"`
	Active *bool   `json:"active"`
}

type TransferOwnershipRequest struct {
	UserID uint `json:"user_id"`
}

type InviteRequest struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

type InvitationDTO struct {
	ID        uint      `json:"id"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	ExpiresAt time.Time `json:"expires_at"`
}

// InvitationInfoDTO is what the accept page shows before the user accepts
type InvitationInfoDTO struct {
	BusinessName  string `json:"business_name"`
	Email         string `json:"email"`
	Role          string `json:"role"`
	AccountExists bool   `json:"account_exists"` // true: log in first; false: choose a name and password
}

type AcceptInvitationRequest struct {
	Name     string `json:"name"`
	Password string `json:"password"`
	Phone    string `json:"phone"`
}

type SupportVisitDTO struct {
	UserID    uint      `json:"user_id"`
	VisitedAt time.Time `json:"visited_at"`
}

// MyBusinessDTO feeds the dashboard's business switcher
type MyBusinessDTO struct {
	BusinessID uint   `json:"business_id"`
	Name       string `json:"name"`
	Slug       string `json:"slug"`
	Status     string `json:"status"`
	Role       string `json:"role"`
	Active     bool   `json:"active"`
}

// ---------- password reset ----------

type ForgotPasswordRequest struct {
	Email string `json:"email"`
}

type ResetPasswordRequest struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}

// ---------- public ----------

type PublicBusinessDTO struct {
	Name            string            `json:"name"`
	Slug            string            `json:"slug"`
	Timezone        string            `json:"timezone"`
	Currency        string            `json:"currency"`
	Language        string            `json:"language"`
	LogoURL         string            `json:"logo_url"`
	CoverURL        string            `json:"cover_url"`
	BrandColor      string            `json:"brand_color"`
	Description     string            `json:"description"`
	Address         string            `json:"address"`
	Phone           string            `json:"phone"`
	MapURL          string            `json:"map_url"`
	SocialLinks     map[string]string `json:"social_links"`
	BookingPolicy   string            `json:"booking_policy"`
	BookingMode     string            `json:"booking_mode"`
	RequireApproval bool              `json:"require_approval"`
}

// PublicCoachDTO leaves out the coach's private phone number
type PublicCoachDTO struct {
	ID       uint          `json:"id"`
	Name     string        `json:"name"`
	Bio      string        `json:"bio"`
	ImageURL string        `json:"image_url"`
	Sports   []SportRefDTO `json:"sports"`
}
