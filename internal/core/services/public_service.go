package services

import (
	"backend/internal/core/domain"
	"backend/internal/core/dto"
	"backend/internal/core/port/input"
	"backend/internal/core/port/output"
	"context"
	"time"
)

type PublicServiceImpl struct {
	businesses output.BusinessRepository
	sports     output.SportRepository
	resources  output.ResourceRepository
	coaches    output.CoachRepository
	classes    output.ClassSessionRepository
	bookings   output.BookingRepository
	files      mediaFiles
	now        func() time.Time
}

func NewPublicService(
	businesses output.BusinessRepository,
	sports output.SportRepository,
	resources output.ResourceRepository,
	coaches output.CoachRepository,
	classes output.ClassSessionRepository,
	bookings output.BookingRepository,
	storage output.FileStorage,
) input.PublicService {
	return &PublicServiceImpl{
		businesses: businesses, sports: sports, resources: resources, coaches: coaches, classes: classes, bookings: bookings,
		files: mediaFiles{storage: storage}, now: time.Now,
	}
}

// published loads a business that may be shown publicly. Drafts, suspended and unknown
// slugs give the same 404, so the page does not reveal that a business exists.
func (s *PublicServiceImpl) published(ctx context.Context, slug string) (*domain.Business, error) {
	return loadPublished(ctx, s.businesses, slug)
}

func loadPublished(ctx context.Context, businesses output.BusinessRepository, slug string) (*domain.Business, error) {
	b, err := businesses.GetBySlug(ctx, slug)
	if err != nil {
		if isNotFound(err) {
			return nil, notFound("BUSINESS_NOT_FOUND")
		}
		return nil, err
	}
	if !b.IsPublic() {
		return nil, notFound("BUSINESS_NOT_FOUND")
	}
	return b, nil
}

// activeOnly lists every active item; plan limits keep these lists small
var activeOnly = output.ListQuery{Filters: map[string]any{"active": true}}

func (s *PublicServiceImpl) Business(ctx context.Context, slug string) (*dto.PublicBusinessDTO, error) {
	b, err := s.published(ctx, slug)
	if err != nil {
		return nil, err
	}
	return &dto.PublicBusinessDTO{
		Name: b.Name, Slug: b.Slug, Timezone: b.Timezone, Currency: b.Currency, Language: b.Language,
		LogoURL: s.files.url(b.LogoKey), CoverURL: s.files.url(b.CoverKey), BrandColor: b.BrandColor,
		Description: b.Description, Address: b.Address, Phone: b.Phone, MapURL: b.MapURL,
		SocialLinks: b.SocialLinks, BookingPolicy: b.BookingPolicy, BookingMode: b.BookingMode,
		RequireApproval: b.RequireApproval,
	}, nil
}

func (s *PublicServiceImpl) Sports(ctx context.Context, slug string) ([]dto.SportDTO, error) {
	b, err := s.published(ctx, slug)
	if err != nil {
		return nil, err
	}
	items, _, err := s.sports.List(ctx, b.ID, activeOnly)
	if err != nil {
		return nil, err
	}
	return mapSlice(items, func(m domain.Sport) dto.SportDTO {
		return dto.SportDTO{
			ID: m.ID, Name: m.Name, Slug: m.Slug, Description: m.Description,
			ImageKey: m.ImageKey, ImageURL: s.files.url(m.ImageKey), Active: m.Active,
		}
	}), nil
}

func (s *PublicServiceImpl) Resources(ctx context.Context, slug string, sportID uint) ([]dto.ResourceDTO, error) {
	b, err := s.published(ctx, slug)
	if err != nil {
		return nil, err
	}
	q := output.ListQuery{Filters: map[string]any{"active": true}}
	if sportID != 0 {
		q.Filters["sport_id"] = sportID
	}
	items, _, err := s.resources.List(ctx, b.ID, q)
	if err != nil {
		return nil, err
	}
	return mapSlice(items, toResourceDTO), nil
}

func (s *PublicServiceImpl) Coaches(ctx context.Context, slug string, sportID uint) ([]dto.PublicCoachDTO, error) {
	b, err := s.published(ctx, slug)
	if err != nil {
		return nil, err
	}
	items, _, err := s.coaches.List(ctx, b.ID, activeOnly)
	if err != nil {
		return nil, err
	}
	out := make([]dto.PublicCoachDTO, 0, len(items))
	for _, c := range items {
		if sportID != 0 && !teaches(c, sportID) {
			continue
		}
		out = append(out, dto.PublicCoachDTO{
			ID: c.ID, Name: c.Name, Bio: c.Bio, ImageURL: s.files.url(c.ImageKey), Sports: sportRefs(c.Sports),
		})
	}
	return out, nil
}

func teaches(c domain.Coach, sportID uint) bool {
	for _, sp := range c.Sports {
		if sp.ID == sportID {
			return true
		}
	}
	return false
}

// ClassSessions lists upcoming active sessions, soonest first, with the seats still free
func (s *PublicServiceImpl) ClassSessions(ctx context.Context, slug string, q dto.ListQuery) (*dto.PagedResult[dto.ClassSessionDTO], error) {
	b, err := s.published(ctx, slug)
	if err != nil {
		return nil, err
	}
	lq, page, limit := newListQuery(q)
	if q.SportID != 0 {
		lq.Filters["sport_id"] = q.SportID
	}
	items, total, err := s.classes.ListUpcoming(ctx, b.ID, s.now(), lq)
	if err != nil {
		return nil, err
	}
	ids := mapSlice(items, func(c domain.ClassSession) uint { return c.ID })
	taken, err := s.bookings.SeatsTaken(ctx, b.ID, ids)
	if err != nil {
		return nil, err
	}
	return pageOf(items, total, page, limit, func(c domain.ClassSession) dto.ClassSessionDTO {
		d := toClassSessionDTO(c)
		left := max(c.Capacity-taken[c.ID], 0)
		d.SeatsLeft = &left
		return d
	}), nil
}
