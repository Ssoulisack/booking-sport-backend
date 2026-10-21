package services

import (
	"backend/internal/core/domain"
	"backend/internal/core/dto"
	"backend/internal/core/port/output"
	"context"
	"net/http"
	"testing"

	"backend/pkg/response"
)

// ---- fakes: only the methods the rule service calls ----

type fakeRuleRepo struct {
	rules map[string]*domain.BookingRule
}

func ruleKey(t string, id uint) string { return t + ":" + string(rune('0'+id)) }

func (f *fakeRuleRepo) GetByScope(_ context.Context, t string, id uint) (*domain.BookingRule, error) {
	return f.rules[ruleKey(t, id)], nil
}
func (f *fakeRuleRepo) Upsert(_ context.Context, r *domain.BookingRule) error {
	f.rules[ruleKey(r.ScopeType, r.ScopeID)] = r
	return nil
}
func (f *fakeRuleRepo) List(context.Context) ([]domain.BookingRule, error) { return nil, nil }
func (f *fakeRuleRepo) DeleteByScope(_ context.Context, t string, id uint) error {
	delete(f.rules, ruleKey(t, id))
	return nil
}

type fakeSports struct {
	output.CRUDRepository[domain.Sport]
	ids map[uint]bool
}

func (f fakeSports) GetByID(_ context.Context, id uint) (*domain.Sport, error) {
	if !f.ids[id] {
		return nil, response.NewError(http.StatusNotFound, "data not found")
	}
	return &domain.Sport{}, nil
}

type fakeResources struct {
	output.CRUDRepository[domain.Resource]
	sportOf map[uint]uint // resource id -> sport id
}

func (f fakeResources) GetByID(_ context.Context, id uint) (*domain.Resource, error) {
	sportID, ok := f.sportOf[id]
	if !ok {
		return nil, response.NewError(http.StatusNotFound, "data not found")
	}
	return &domain.Resource{SportID: sportID}, nil
}

const (
	tennis, soccer, boxing = 1, 2, 3
	tennisCourt1           = 1
	soccerPitch1           = 2
)

func ip(n int) *int { return &n }

// newRuleService mirrors the seeded default: min 60, max 180, step 30
func newRuleService() (*RuleServiceImpl, *fakeRuleRepo) {
	repo := &fakeRuleRepo{rules: map[string]*domain.BookingRule{}}
	repo.rules[ruleKey(domain.ScopeGlobal, 0)] = &domain.BookingRule{
		ScopeType: domain.ScopeGlobal, MinDurationMin: ip(60), MaxDurationMin: ip(180),
		DurationStepMin: ip(30), StartStepMin: ip(30), CancelBeforeHours: ip(24),
		MaxDaysAhead: ip(30), MinNoticeMin: ip(60), Capacity: ip(10),
	}
	svc := &RuleServiceImpl{
		rules:     repo,
		sports:    fakeSports{ids: map[uint]bool{tennis: true, soccer: true, boxing: true}},
		resources: fakeResources{sportOf: map[uint]uint{tennisCourt1: tennis, soccerPitch1: soccer}},
	}
	return svc, repo
}

func mustSet(t *testing.T, svc *RuleServiceImpl, scope string, id uint, req dto.RuleRequest) {
	t.Helper()
	if _, err := svc.SetRule(context.Background(), scope, id, req); err != nil {
		t.Fatalf("SetRule(%s,%d): %v", scope, id, err)
	}
}

func TestEffectiveSportRules(t *testing.T) {
	svc, _ := newRuleService()
	ctx := context.Background()

	// tennis: admin-chosen maximum 240; soccer: max 90; boxing: fixed 60-minute session
	mustSet(t, svc, domain.ScopeSport, tennis, dto.RuleRequest{MaxDurationMin: ip(240)})
	mustSet(t, svc, domain.ScopeSport, soccer, dto.RuleRequest{MaxDurationMin: ip(90)})
	mustSet(t, svc, domain.ScopeSport, boxing, dto.RuleRequest{MaxDurationMin: ip(60)})

	cases := []struct {
		name               string
		sportID, resource  uint
		wantMin, wantMax   int
		wantStep, wantCanc int
	}{
		{"tennis court inherits the tennis max", 0, tennisCourt1, 60, 240, 30, 24},
		{"tennis by sport id", tennis, 0, 60, 240, 30, 24},
		{"soccer max 90", soccer, 0, 60, 90, 30, 24},
		{"soccer pitch via resource", 0, soccerPitch1, 60, 90, 30, 24},
		{"boxing is a single 60 minute session", boxing, 0, 60, 60, 30, 24},
		{"no sport or resource gives the global default", 0, 0, 60, 180, 30, 24},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			e, err := svc.Effective(ctx, c.sportID, c.resource)
			if err != nil {
				t.Fatal(err)
			}
			if e.MinDurationMin != c.wantMin || e.MaxDurationMin != c.wantMax ||
				e.DurationStepMin != c.wantStep || e.CancelBeforeHours != c.wantCanc {
				t.Fatalf("got %+v", *e)
			}
		})
	}
}

func TestResourceRuleOverridesSport(t *testing.T) {
	svc, _ := newRuleService()
	mustSet(t, svc, domain.ScopeSport, tennis, dto.RuleRequest{MaxDurationMin: ip(240)})
	mustSet(t, svc, domain.ScopeResource, tennisCourt1, dto.RuleRequest{MaxDurationMin: ip(120), CancelBeforeHours: ip(2)})

	e, err := svc.Effective(context.Background(), 0, tennisCourt1)
	if err != nil {
		t.Fatal(err)
	}
	if e.MaxDurationMin != 120 || e.CancelBeforeHours != 2 || e.MinDurationMin != 60 {
		t.Fatalf("got %+v", *e)
	}
}

func TestSetRuleRejectsInvalid(t *testing.T) {
	svc, _ := newRuleService()
	ctx := context.Background()
	cases := map[string]struct {
		scope string
		id    uint
		req   dto.RuleRequest
		code  string
	}{
		"max below min":                   {domain.ScopeSport, tennis, dto.RuleRequest{MaxDurationMin: ip(30)}, "MAX_DURATION_MUST_NOT_BE_BELOW_MIN"},
		"max is not min plus whole steps": {domain.ScopeSport, tennis, dto.RuleRequest{MaxDurationMin: ip(100)}, "MAX_DURATION_MUST_BE_MIN_PLUS_WHOLE_STEPS"},
		"zero step":                       {domain.ScopeSport, tennis, dto.RuleRequest{DurationStepMin: ip(0)}, "DURATIONS_AND_STEPS_MUST_BE_POSITIVE"},
		"negative cancel window":          {domain.ScopeSport, tennis, dto.RuleRequest{CancelBeforeHours: ip(-1)}, "CANCEL_AND_NOTICE_MUST_NOT_BE_NEGATIVE"},
		"unknown sport":                   {domain.ScopeSport, 9, dto.RuleRequest{MaxDurationMin: ip(90)}, "SPORT_NOT_FOUND"},
		"unknown resource":                {domain.ScopeResource, 9, dto.RuleRequest{MaxDurationMin: ip(90)}, "RESOURCE_NOT_FOUND"},
		"bad scope":                       {"venue", 0, dto.RuleRequest{}, "INVALID_SCOPE"},
		"global needs every field":        {domain.ScopeGlobal, 0, dto.RuleRequest{MaxDurationMin: ip(90)}, "GLOBAL_RULE_NEEDS_EVERY_FIELD"},
		"global takes no id":              {domain.ScopeGlobal, 1, dto.RuleRequest{}, "GLOBAL_RULE_HAS_NO_ID"},
		"resource rule breaks the chain":  {domain.ScopeResource, tennisCourt1, dto.RuleRequest{MinDurationMin: ip(100)}, "MAX_DURATION_MUST_BE_MIN_PLUS_WHOLE_STEPS"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := svc.SetRule(ctx, tc.scope, tc.id, tc.req)
			if err == nil || err.Error() != tc.code {
				t.Fatalf("got %v, want %s", err, tc.code)
			}
		})
	}
}

func TestInvalidRuleIsNotSaved(t *testing.T) {
	svc, repo := newRuleService()
	_, _ = svc.SetRule(context.Background(), domain.ScopeSport, tennis, dto.RuleRequest{MaxDurationMin: ip(30)})
	if repo.rules[ruleKey(domain.ScopeSport, tennis)] != nil {
		t.Fatal("a rejected rule must not be stored")
	}
}

func TestGlobalRuleCannotBeDeleted(t *testing.T) {
	svc, _ := newRuleService()
	if err := svc.DeleteRule(context.Background(), domain.ScopeGlobal, 0); err == nil {
		t.Fatal("global rule must not be deletable")
	}
}

func TestSportRuleMakesNarrowerRuleInconsistent(t *testing.T) {
	svc, _ := newRuleService()
	ctx := context.Background()
	mustSet(t, svc, domain.ScopeResource, tennisCourt1, dto.RuleRequest{MaxDurationMin: ip(120)})
	// raising the sport minimum later breaks the court rule; Effective must refuse to serve it
	mustSet(t, svc, domain.ScopeSport, tennis, dto.RuleRequest{MinDurationMin: ip(150), MaxDurationMin: ip(240)})
	if _, err := svc.Effective(ctx, 0, tennisCourt1); err == nil {
		t.Fatal("expected RULES_ARE_INCONSISTENT")
	}
}
