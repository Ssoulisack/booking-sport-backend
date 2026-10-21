package postgres

import (
	"backend/internal/adapters/secondary/postgres/exception"
	"backend/internal/core/domain"
	"backend/internal/core/port/output"
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
)

// firstOrNil maps "not found" to (nil, nil) for lookups where absence is a normal answer
func firstOrNil[T any](tx *gorm.DB) (*T, error) {
	var m T
	err := tx.First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, exception.HandleGormError(err)
	}
	return &m, nil
}

// ---------- plans ----------

type planRepository struct{ db *gorm.DB }

func NewPlanRepository(db *gorm.DB) output.PlanRepository { return &planRepository{db: db} }

func (r *planRepository) List(ctx context.Context) ([]domain.Plan, error) {
	var plans []domain.Plan
	err := r.db.WithContext(ctx).Order("id ASC").Find(&plans).Error
	return plans, exception.HandleGormError(err)
}

func (r *planRepository) GetByID(ctx context.Context, id uint) (*domain.Plan, error) {
	var p domain.Plan
	if err := r.db.WithContext(ctx).First(&p, id).Error; err != nil {
		return nil, exception.HandleGormError(err)
	}
	return &p, nil
}

func (r *planRepository) Update(ctx context.Context, p *domain.Plan) error {
	return exception.HandleGormError(r.db.WithContext(ctx).Save(p).Error)
}

// ---------- businesses ----------

type businessRepository struct{ db *gorm.DB }

func NewBusinessRepository(db *gorm.DB) output.BusinessRepository { return &businessRepository{db: db} }

func (r *businessRepository) CreateWithDefaults(ctx context.Context, b *domain.Business, rule domain.BookingRule) error {
	return exception.HandleGormError(r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Omit("Plan").Create(b).Error; err != nil {
			return err
		}
		rule.BusinessID = b.ID
		return tx.Create(&rule).Error
	}))
}

func (r *businessRepository) GetByID(ctx context.Context, id uint) (*domain.Business, error) {
	var b domain.Business
	if err := r.db.WithContext(ctx).Preload("Plan").First(&b, id).Error; err != nil {
		return nil, exception.HandleGormError(err)
	}
	return &b, nil
}

func (r *businessRepository) GetBySlug(ctx context.Context, slug string) (*domain.Business, error) {
	var b domain.Business
	if err := r.db.WithContext(ctx).Preload("Plan").Where("slug = ?", slug).First(&b).Error; err != nil {
		return nil, exception.HandleGormError(err)
	}
	return &b, nil
}

func (r *businessRepository) List(ctx context.Context, q output.ListQuery) ([]domain.Business, int64, error) {
	tx := r.db.WithContext(ctx).Model(&domain.Business{})
	if q.Keyword != "" {
		like := "%" + q.Keyword + "%"
		tx = tx.Where("name ILIKE ? OR slug ILIKE ?", like, like)
	}
	tx = applyFilters(tx, q.Filters)
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, exception.HandleGormError(err)
	}
	var items []domain.Business
	err := paginate(tx.Preload("Plan").Order("id ASC"), q).Find(&items).Error
	return items, total, exception.HandleGormError(err)
}

func (r *businessRepository) Update(ctx context.Context, b *domain.Business) error {
	return exception.HandleGormError(r.db.WithContext(ctx).Omit("Plan").Save(b).Error)
}

func (r *businessRepository) SlugTaken(ctx context.Context, slug string, exceptID uint) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&domain.Business{}).Where("slug = ? AND id <> ?", slug, exceptID).Count(&count).Error
	return count > 0, exception.HandleGormError(err)
}

// ---------- memberships ----------

type membershipRepository struct{ db *gorm.DB }

func NewMembershipRepository(db *gorm.DB) output.MembershipRepository {
	return &membershipRepository{db: db}
}

func (r *membershipRepository) Create(ctx context.Context, m *domain.Membership) error {
	return exception.HandleGormError(r.db.WithContext(ctx).Omit("User", "Business").Create(m).Error)
}

func (r *membershipRepository) Get(ctx context.Context, businessID, userID uint) (*domain.Membership, error) {
	return firstOrNil[domain.Membership](r.db.WithContext(ctx).Where("business_id = ? AND user_id = ?", businessID, userID))
}

func (r *membershipRepository) ListByBusiness(ctx context.Context, businessID uint) ([]domain.Membership, error) {
	var items []domain.Membership
	err := r.db.WithContext(ctx).Preload("User").Where("business_id = ?", businessID).Order("id ASC").Find(&items).Error
	return items, exception.HandleGormError(err)
}

func (r *membershipRepository) ListByUser(ctx context.Context, userID uint) ([]domain.Membership, error) {
	var items []domain.Membership
	err := r.db.WithContext(ctx).Preload("Business").Where("user_id = ?", userID).Order("id ASC").Find(&items).Error
	return items, exception.HandleGormError(err)
}

func (r *membershipRepository) Update(ctx context.Context, m *domain.Membership) error {
	return exception.HandleGormError(r.db.WithContext(ctx).Omit("User", "Business").Save(m).Error)
}

func (r *membershipRepository) Delete(ctx context.Context, businessID, userID uint) error {
	return requireAffected(r.db.WithContext(ctx).
		Where("business_id = ? AND user_id = ?", businessID, userID).Delete(&domain.Membership{}))
}

func (r *membershipRepository) Count(ctx context.Context, businessID uint) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&domain.Membership{}).Where("business_id = ?", businessID).Count(&count).Error
	return count, exception.HandleGormError(err)
}

func (r *membershipRepository) HasOwner(ctx context.Context, businessID uint) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&domain.Membership{}).
		Where("business_id = ? AND role = ?", businessID, domain.MemberOwner).Count(&count).Error
	return count > 0, exception.HandleGormError(err)
}

func (r *membershipRepository) TransferOwnership(ctx context.Context, businessID, fromUserID, toUserID uint) error {
	return exception.HandleGormError(r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&domain.Membership{}).
			Where("business_id = ? AND user_id = ? AND role = ?", businessID, fromUserID, domain.MemberOwner).
			Update("role", domain.MemberManager)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected != 1 {
			return gorm.ErrRecordNotFound
		}
		res = tx.Model(&domain.Membership{}).
			Where("business_id = ? AND user_id = ?", businessID, toUserID).
			Updates(map[string]any{"role": domain.MemberOwner, "active": true})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected != 1 {
			return gorm.ErrRecordNotFound
		}
		return nil
	}))
}

// ---------- invitations ----------

type invitationRepository struct{ db *gorm.DB }

func NewInvitationRepository(db *gorm.DB) output.InvitationRepository {
	return &invitationRepository{db: db}
}

func (r *invitationRepository) pending(ctx context.Context, businessID uint, now time.Time) *gorm.DB {
	return r.db.WithContext(ctx).Model(&domain.Invitation{}).
		Where("business_id = ? AND accepted_at IS NULL AND expires_at > ?", businessID, now)
}

func (r *invitationRepository) Create(ctx context.Context, inv *domain.Invitation) error {
	return exception.HandleGormError(r.db.WithContext(ctx).Create(inv).Error)
}

func (r *invitationRepository) GetByTokenHash(ctx context.Context, hash string) (*domain.Invitation, error) {
	return firstOrNil[domain.Invitation](r.db.WithContext(ctx).Where("token_hash = ?", hash))
}

func (r *invitationRepository) GetByID(ctx context.Context, businessID, id uint) (*domain.Invitation, error) {
	var inv domain.Invitation
	if err := r.db.WithContext(ctx).Where("business_id = ?", businessID).First(&inv, id).Error; err != nil {
		return nil, exception.HandleGormError(err)
	}
	return &inv, nil
}

func (r *invitationRepository) ListPending(ctx context.Context, businessID uint, now time.Time) ([]domain.Invitation, error) {
	var items []domain.Invitation
	err := r.pending(ctx, businessID, now).Order("id ASC").Find(&items).Error
	return items, exception.HandleGormError(err)
}

func (r *invitationRepository) CountPending(ctx context.Context, businessID uint, now time.Time) (int64, error) {
	var count int64
	err := r.pending(ctx, businessID, now).Count(&count).Error
	return count, exception.HandleGormError(err)
}

func (r *invitationRepository) PendingForEmail(ctx context.Context, businessID uint, email string, now time.Time) (*domain.Invitation, error) {
	return firstOrNil[domain.Invitation](r.pending(ctx, businessID, now).Where("email = ?", email))
}

func (r *invitationRepository) PendingOwnerInvite(ctx context.Context, businessID uint, now time.Time) (*domain.Invitation, error) {
	return firstOrNil[domain.Invitation](r.pending(ctx, businessID, now).Where("role = ?", domain.MemberOwner))
}

func (r *invitationRepository) Update(ctx context.Context, inv *domain.Invitation) error {
	return exception.HandleGormError(r.db.WithContext(ctx).Save(inv).Error)
}

func (r *invitationRepository) Delete(ctx context.Context, businessID, id uint) error {
	return requireAffected(r.db.WithContext(ctx).Where("business_id = ?", businessID).Delete(&domain.Invitation{}, id))
}

// ---------- password resets ----------

type passwordResetRepository struct{ db *gorm.DB }

func NewPasswordResetRepository(db *gorm.DB) output.PasswordResetRepository {
	return &passwordResetRepository{db: db}
}

func (r *passwordResetRepository) Create(ctx context.Context, reset *domain.PasswordReset) error {
	return exception.HandleGormError(r.db.WithContext(ctx).Create(reset).Error)
}

func (r *passwordResetRepository) GetByTokenHash(ctx context.Context, hash string) (*domain.PasswordReset, error) {
	return firstOrNil[domain.PasswordReset](r.db.WithContext(ctx).Where("token_hash = ?", hash))
}

func (r *passwordResetRepository) UseAll(ctx context.Context, userID uint, at time.Time) error {
	return exception.HandleGormError(r.db.WithContext(ctx).Model(&domain.PasswordReset{}).
		Where("user_id = ? AND used_at IS NULL", userID).Update("used_at", at).Error)
}

// ---------- support visits ----------

type supportVisitRepository struct{ db *gorm.DB }

func NewSupportVisitRepository(db *gorm.DB) output.SupportVisitRepository {
	return &supportVisitRepository{db: db}
}

func (r *supportVisitRepository) Record(ctx context.Context, userID, businessID uint, at time.Time, window time.Duration) error {
	var recent int64
	err := r.db.WithContext(ctx).Model(&domain.SupportVisit{}).
		Where("user_id = ? AND business_id = ? AND visited_at > ?", userID, businessID, at.Add(-window)).
		Count(&recent).Error
	if err != nil {
		return exception.HandleGormError(err)
	}
	if recent > 0 {
		return nil
	}
	visit := domain.SupportVisit{UserID: userID, BusinessID: businessID, VisitedAt: at}
	return exception.HandleGormError(r.db.WithContext(ctx).Create(&visit).Error)
}

func (r *supportVisitRepository) List(ctx context.Context, businessID uint, q output.ListQuery) ([]domain.SupportVisit, int64, error) {
	tx := r.db.WithContext(ctx).Model(&domain.SupportVisit{}).Where("business_id = ?", businessID)
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, exception.HandleGormError(err)
	}
	var items []domain.SupportVisit
	err := paginate(tx.Order("visited_at DESC"), q).Find(&items).Error
	return items, total, exception.HandleGormError(err)
}
