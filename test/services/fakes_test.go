// Package servicetest tests the core services from the outside: only their public constructors
// and methods, with in-memory fakes for the repositories. Run with: go test ./test/services
package servicetest

import (
	"backend/internal/core/domain"
	"backend/internal/core/port/output"
	"backend/pkg/response"
	"context"
	"fmt"
	"io"
	"net/http"
)

var errNotFound = response.NewError(http.StatusNotFound, "data not found")

func ip(n int) *int { return &n }

// ---------- file storage ----------

type fakeStorage struct {
	uploaded map[string][]byte
	deleted  []string
}

func newFakeStorage() *fakeStorage { return &fakeStorage{uploaded: map[string][]byte{}} }

func (f *fakeStorage) Upload(_ context.Context, key, _ string, body io.Reader) error {
	b, _ := io.ReadAll(body)
	f.uploaded[key] = b
	return nil
}
func (f *fakeStorage) Delete(_ context.Context, key string) error {
	f.deleted = append(f.deleted, key)
	return nil
}
func (f *fakeStorage) URL(key string) string { return "https://files.test/" + key }

// ---------- media records ----------

type fakeMedia struct{ items map[string]domain.Media }

func newFakeMedia() *fakeMedia { return &fakeMedia{items: map[string]domain.Media{}} }

func (f *fakeMedia) Create(_ context.Context, m *domain.Media) error {
	f.items[m.Key] = *m
	return nil
}
func (f *fakeMedia) Get(_ context.Context, businessID uint, key string) (*domain.Media, error) {
	m, ok := f.items[key]
	if !ok || m.BusinessID != businessID {
		return nil, nil
	}
	return &m, nil
}
func (f *fakeMedia) Delete(_ context.Context, _ uint, key string) error {
	delete(f.items, key)
	return nil
}
func (f *fakeMedia) Usage(_ context.Context, businessID uint) (int64, int64, error) {
	var n, size int64
	for _, m := range f.items {
		if m.BusinessID == businessID {
			n, size = n+1, size+m.SizeBytes
		}
	}
	return n, size, nil
}

// ---------- businesses (GetByID only) ----------

type fakeBusinesses struct {
	output.BusinessRepository
	limits domain.PlanLimits
}

func (f fakeBusinesses) GetByID(_ context.Context, id uint) (*domain.Business, error) {
	b := domain.Business{Timezone: "Asia/Vientiane", Plan: domain.Plan{Name: "Test", Limits: f.limits}}
	b.ID = id
	return &b, nil
}

// ---------- sports and resources: ids belong to one business ----------

type fakeSports struct {
	output.ScopedRepository[domain.Sport]
	owner   map[uint]uint // sport id -> business id
	updated []domain.Sport
}

func (f *fakeSports) GetByID(_ context.Context, businessID, id uint) (*domain.Sport, error) {
	if f.owner[id] != businessID {
		return nil, errNotFound
	}
	s := domain.Sport{BusinessID: businessID, Name: "Tennis", Slug: "tennis", Active: true}
	s.ID = id
	return &s, nil
}
func (f *fakeSports) Update(_ context.Context, m *domain.Sport) error {
	f.updated = append(f.updated, *m)
	return nil
}

type fakeResources struct {
	output.ScopedRepository[domain.Resource]
	sportOf map[uint]uint // resource id -> sport id
	owner   map[uint]uint // resource id -> business id
}

func (f fakeResources) GetByID(_ context.Context, businessID, id uint) (*domain.Resource, error) {
	if f.owner[id] != businessID {
		return nil, errNotFound
	}
	r := domain.Resource{BusinessID: businessID, SportID: f.sportOf[id], Active: true}
	r.ID = id
	return &r, nil
}

// ---------- booking rules ----------

type fakeRuleRepo struct {
	rules map[string]*domain.BookingRule
}

func ruleKey(businessID uint, t string, id uint) string {
	return fmt.Sprintf("%d:%s:%d", businessID, t, id)
}

func (f *fakeRuleRepo) GetByScope(_ context.Context, b uint, t string, id uint) (*domain.BookingRule, error) {
	return f.rules[ruleKey(b, t, id)], nil
}
func (f *fakeRuleRepo) Upsert(_ context.Context, r *domain.BookingRule) error {
	f.rules[ruleKey(r.BusinessID, r.ScopeType, r.ScopeID)] = r
	return nil
}
func (f *fakeRuleRepo) List(context.Context, uint) ([]domain.BookingRule, error) { return nil, nil }
func (f *fakeRuleRepo) DeleteByScope(_ context.Context, b uint, t string, id uint) error {
	delete(f.rules, ruleKey(b, t, id))
	return nil
}

// ---------- opening hours ----------

type fakeHours struct {
	rows map[string][]domain.OpeningHour
}

func newFakeHours() *fakeHours { return &fakeHours{rows: map[string][]domain.OpeningHour{}} }

func hoursKey(b uint, t string, id uint) string { return fmt.Sprintf("%d:%s:%d", b, t, id) }

func (f *fakeHours) ListByOwner(_ context.Context, b uint, t string, id uint) ([]domain.OpeningHour, error) {
	return f.rows[hoursKey(b, t, id)], nil
}
func (f *fakeHours) ReplaceForOwner(_ context.Context, b uint, t string, id uint, hours []domain.OpeningHour) error {
	f.rows[hoursKey(b, t, id)] = hours
	return nil
}
