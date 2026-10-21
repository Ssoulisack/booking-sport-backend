package services

import (
	"backend/internal/core/domain"
	"backend/internal/core/dto"
	"backend/internal/core/port/input"
	"backend/internal/core/port/output"
	"context"
	"log"
	"strings"
)

type CatalogServiceImpl struct {
	sports    output.SportRepository
	resources output.ResourceRepository
	coaches   output.CoachRepository
	storage   output.FileStorage
}

func NewCatalogService(
	sports output.SportRepository,
	resources output.ResourceRepository,
	coaches output.CoachRepository,
	storage output.FileStorage,
) input.CatalogService {
	return &CatalogServiceImpl{sports: sports, resources: resources, coaches: coaches, storage: storage}
}

var resourceKinds = map[string]bool{
	domain.ResourceKindCourt: true,
	domain.ResourceKindPitch: true,
	domain.ResourceKindRing:  true,
	domain.ResourceKindOther: true,
}

// ---------- sports ----------

func (s *CatalogServiceImpl) toSportDTO(m domain.Sport) dto.SportDTO {
	return dto.SportDTO{
		ID: m.ID, Name: m.Name, Slug: m.Slug, Description: m.Description,
		ImageKey: m.ImageKey, ImageURL: s.storage.URL(m.ImageKey), Active: m.Active,
	}
}

func validateSport(req *dto.SportRequest) error {
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		return badRequest("NAME_IS_REQUIRED")
	}
	if req.ImageKey != "" && !isValidMediaKey(req.ImageKey) {
		return badRequest("INVALID_FILE_KEY")
	}
	return nil
}

func (s *CatalogServiceImpl) CreateSport(ctx context.Context, req dto.SportRequest) (*dto.SportDTO, error) {
	if err := validateSport(&req); err != nil {
		return nil, err
	}
	slug := slugify(req.Name)
	existing, _, err := s.sports.List(ctx, output.ListQuery{Page: 1, Limit: 1, Filters: map[string]any{"slug": slug}})
	if err != nil {
		return nil, err
	}
	if len(existing) > 0 {
		return nil, conflict("SPORT_ALREADY_EXISTS")
	}
	m := domain.Sport{
		Name: req.Name, Slug: slug, Description: req.Description,
		ImageKey: req.ImageKey, Active: isActive(req.Active),
	}
	if err := s.sports.Create(ctx, &m); err != nil {
		return nil, err
	}
	d := s.toSportDTO(m)
	return &d, nil
}

func (s *CatalogServiceImpl) GetSport(ctx context.Context, id uint) (*dto.SportDTO, error) {
	m, err := s.sports.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	d := s.toSportDTO(*m)
	return &d, nil
}

func (s *CatalogServiceImpl) ListSports(ctx context.Context, q dto.ListQuery) (*dto.PagedResult[dto.SportDTO], error) {
	lq, page, limit := newListQuery(q)
	if q.Active != nil {
		lq.Filters["active"] = *q.Active
	}
	items, total, err := s.sports.List(ctx, lq)
	if err != nil {
		return nil, err
	}
	return pageOf(items, total, page, limit, s.toSportDTO), nil
}

// UpdateSport keeps the slug stable so public URLs do not change when a sport is renamed
func (s *CatalogServiceImpl) UpdateSport(ctx context.Context, id uint, req dto.SportRequest) (*dto.SportDTO, error) {
	if err := validateSport(&req); err != nil {
		return nil, err
	}
	m, err := s.sports.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	oldKey := m.ImageKey
	m.Name, m.Description, m.ImageKey, m.Active = req.Name, req.Description, req.ImageKey, isActive(req.Active)
	if err := s.sports.Update(ctx, m); err != nil {
		return nil, err
	}
	s.removeReplacedImage(ctx, oldKey, m.ImageKey)
	d := s.toSportDTO(*m)
	return &d, nil
}

func (s *CatalogServiceImpl) DeleteSport(ctx context.Context, id uint) error {
	m, err := s.sports.GetByID(ctx, id)
	if err != nil {
		return err
	}
	inUse, _, err := s.resources.List(ctx, output.ListQuery{Page: 1, Limit: 1, Filters: map[string]any{"sport_id": id}})
	if err != nil {
		return err
	}
	if len(inUse) > 0 {
		return conflict("SPORT_HAS_RESOURCES")
	}
	if err := s.sports.Delete(ctx, id); err != nil {
		return err
	}
	s.removeReplacedImage(ctx, m.ImageKey, "")
	return nil
}

// ---------- resources ----------

func toResourceDTO(m domain.Resource) dto.ResourceDTO {
	return dto.ResourceDTO{ID: m.ID, SportID: m.SportID, Name: m.Name, Kind: m.Kind, Active: m.Active}
}

func (s *CatalogServiceImpl) validateResource(ctx context.Context, req *dto.ResourceRequest) error {
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		return badRequest("NAME_IS_REQUIRED")
	}
	if !resourceKinds[req.Kind] {
		return badRequest("INVALID_RESOURCE_KIND")
	}
	_, err := s.sports.GetByID(ctx, req.SportID)
	return orBadRequest(err, "SPORT_NOT_FOUND")
}

func (s *CatalogServiceImpl) CreateResource(ctx context.Context, req dto.ResourceRequest) (*dto.ResourceDTO, error) {
	if err := s.validateResource(ctx, &req); err != nil {
		return nil, err
	}
	m := domain.Resource{SportID: req.SportID, Name: req.Name, Kind: req.Kind, Active: isActive(req.Active)}
	if err := s.resources.Create(ctx, &m); err != nil {
		return nil, err
	}
	d := toResourceDTO(m)
	return &d, nil
}

func (s *CatalogServiceImpl) GetResource(ctx context.Context, id uint) (*dto.ResourceDTO, error) {
	m, err := s.resources.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	d := toResourceDTO(*m)
	return &d, nil
}

func (s *CatalogServiceImpl) ListResources(ctx context.Context, q dto.ListQuery) (*dto.PagedResult[dto.ResourceDTO], error) {
	lq, page, limit := newListQuery(q)
	if q.SportID != 0 {
		lq.Filters["sport_id"] = q.SportID
	}
	if q.Active != nil {
		lq.Filters["active"] = *q.Active
	}
	items, total, err := s.resources.List(ctx, lq)
	if err != nil {
		return nil, err
	}
	return pageOf(items, total, page, limit, toResourceDTO), nil
}

func (s *CatalogServiceImpl) UpdateResource(ctx context.Context, id uint, req dto.ResourceRequest) (*dto.ResourceDTO, error) {
	if err := s.validateResource(ctx, &req); err != nil {
		return nil, err
	}
	m, err := s.resources.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	m.SportID, m.Name, m.Kind, m.Active = req.SportID, req.Name, req.Kind, isActive(req.Active)
	if err := s.resources.Update(ctx, m); err != nil {
		return nil, err
	}
	d := toResourceDTO(*m)
	return &d, nil
}

// DeleteResource will also have to refuse resources with future bookings once bookings exist (phase 3)
func (s *CatalogServiceImpl) DeleteResource(ctx context.Context, id uint) error {
	return s.resources.Delete(ctx, id)
}

// ---------- coaches ----------

func (s *CatalogServiceImpl) toCoachDTO(m domain.Coach) dto.CoachDTO {
	sports := make([]dto.SportRefDTO, 0, len(m.Sports))
	for _, sp := range m.Sports {
		sports = append(sports, dto.SportRefDTO{ID: sp.ID, Name: sp.Name})
	}
	return dto.CoachDTO{
		ID: m.ID, Name: m.Name, Bio: m.Bio, Phone: m.Phone, ImageKey: m.ImageKey,
		ImageURL: s.storage.URL(m.ImageKey), Active: m.Active, Sports: sports,
	}
}

func validateCoach(req *dto.CoachRequest) error {
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		return badRequest("NAME_IS_REQUIRED")
	}
	if req.ImageKey != "" && !isValidMediaKey(req.ImageKey) {
		return badRequest("INVALID_FILE_KEY")
	}
	return nil
}

func (s *CatalogServiceImpl) CreateCoach(ctx context.Context, req dto.CoachRequest) (*dto.CoachDTO, error) {
	if err := validateCoach(&req); err != nil {
		return nil, err
	}
	m := domain.Coach{
		Name: req.Name, Bio: req.Bio, Phone: strings.TrimSpace(req.Phone),
		ImageKey: req.ImageKey, Active: isActive(req.Active),
	}
	if err := s.coaches.Create(ctx, &m); err != nil {
		return nil, err
	}
	if err := s.coaches.SetSports(ctx, m.ID, req.SportIDs); err != nil {
		// do not leave a coach whose sports could not be saved
		if delErr := s.coaches.Delete(ctx, m.ID); delErr != nil {
			log.Printf("❌ could not roll back coach %d: %v", m.ID, delErr)
		}
		return nil, err
	}
	return s.GetCoach(ctx, m.ID)
}

func (s *CatalogServiceImpl) GetCoach(ctx context.Context, id uint) (*dto.CoachDTO, error) {
	m, err := s.coaches.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	d := s.toCoachDTO(*m)
	return &d, nil
}

func (s *CatalogServiceImpl) ListCoaches(ctx context.Context, q dto.ListQuery) (*dto.PagedResult[dto.CoachDTO], error) {
	lq, page, limit := newListQuery(q)
	if q.Active != nil {
		lq.Filters["active"] = *q.Active
	}
	items, total, err := s.coaches.List(ctx, lq)
	if err != nil {
		return nil, err
	}
	return pageOf(items, total, page, limit, s.toCoachDTO), nil
}

func (s *CatalogServiceImpl) UpdateCoach(ctx context.Context, id uint, req dto.CoachRequest) (*dto.CoachDTO, error) {
	if err := validateCoach(&req); err != nil {
		return nil, err
	}
	m, err := s.coaches.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	// check the sports first so a bad id does not leave a half-updated coach
	if err := s.coaches.SetSports(ctx, id, req.SportIDs); err != nil {
		return nil, err
	}
	oldKey := m.ImageKey
	m.Name, m.Bio, m.Phone = req.Name, req.Bio, strings.TrimSpace(req.Phone)
	m.ImageKey, m.Active = req.ImageKey, isActive(req.Active)
	m.Sports = nil // SetSports owns the association
	if err := s.coaches.Update(ctx, m); err != nil {
		return nil, err
	}
	s.removeReplacedImage(ctx, oldKey, m.ImageKey)
	return s.GetCoach(ctx, id)
}

func (s *CatalogServiceImpl) DeleteCoach(ctx context.Context, id uint) error {
	m, err := s.coaches.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if err := s.coaches.Delete(ctx, id); err != nil {
		return err
	}
	s.removeReplacedImage(ctx, m.ImageKey, "")
	return nil
}

// removeReplacedImage deletes the old file after a successful save; a failure only leaves an orphan file
func (s *CatalogServiceImpl) removeReplacedImage(ctx context.Context, oldKey, newKey string) {
	if oldKey == "" || oldKey == newKey {
		return
	}
	if err := s.storage.Delete(ctx, oldKey); err != nil {
		log.Printf("⚠️  could not delete replaced image %s: %v", oldKey, err)
	}
}
