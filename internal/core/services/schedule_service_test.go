package services

import (
	"backend/internal/core/dto"
	"testing"
)

func hours(weekday int, open, close string) dto.OpeningHourDTO {
	return dto.OpeningHourDTO{Weekday: weekday, Open: open, Close: close}
}

func TestBuildOpeningHoursValid(t *testing.T) {
	cases := map[string][]dto.OpeningHourDTO{
		"full week":                  {hours(1, "08:00", "22:00"), hours(2, "08:00", "22:00")},
		"split day":                  {hours(1, "08:00", "12:00"), hours(1, "13:00", "22:00")},
		"touching intervals":         {hours(1, "08:00", "12:00"), hours(1, "12:00", "22:00")},
		"closes at midnight":         {hours(5, "10:00", "24:00")},
		"unsorted input":             {hours(1, "13:00", "22:00"), hours(1, "08:00", "12:00")},
		"never open (empty)":         {},
		"sunday is 0, saturday is 6": {hours(0, "09:00", "18:00"), hours(6, "09:00", "18:00")},
	}
	for name, in := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := buildOpeningHours("resource", 1, in); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestBuildOpeningHoursInvalid(t *testing.T) {
	cases := map[string]struct {
		in   []dto.OpeningHourDTO
		code string
	}{
		"close before open":    {[]dto.OpeningHourDTO{hours(1, "22:00", "08:00")}, "CLOSE_MUST_BE_AFTER_OPEN"},
		"close equals open":    {[]dto.OpeningHourDTO{hours(1, "08:00", "08:00")}, "CLOSE_MUST_BE_AFTER_OPEN"},
		"overlap":              {[]dto.OpeningHourDTO{hours(1, "08:00", "14:00"), hours(1, "13:00", "22:00")}, "OPENING_HOURS_OVERLAP"},
		"weekday too big":      {[]dto.OpeningHourDTO{hours(7, "08:00", "22:00")}, "INVALID_WEEKDAY"},
		"negative weekday":     {[]dto.OpeningHourDTO{hours(-1, "08:00", "22:00")}, "INVALID_WEEKDAY"},
		"bad format":           {[]dto.OpeningHourDTO{hours(1, "8:00", "22:00")}, "INVALID_TIME_FORMAT"},
		"minutes out of range": {[]dto.OpeningHourDTO{hours(1, "08:60", "22:00")}, "INVALID_TIME_FORMAT"},
		"hour out of range":    {[]dto.OpeningHourDTO{hours(1, "08:00", "25:00")}, "INVALID_TIME_FORMAT"},
		"24:00 cannot open":    {[]dto.OpeningHourDTO{hours(1, "24:00", "24:00")}, "INVALID_TIME_FORMAT"},
		"text":                 {[]dto.OpeningHourDTO{hours(1, "morning", "evening")}, "INVALID_TIME_FORMAT"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := buildOpeningHours("resource", 1, tc.in)
			if err == nil || err.Error() != tc.code {
				t.Fatalf("got %v, want %s", err, tc.code)
			}
		})
	}
}

func TestOverlapIsPerWeekday(t *testing.T) {
	in := []dto.OpeningHourDTO{hours(1, "08:00", "22:00"), hours(2, "08:00", "22:00")}
	if _, err := buildOpeningHours("coach", 3, in); err != nil {
		t.Fatalf("same hours on different days must not overlap: %v", err)
	}
}

func TestBuildOpeningHoursNormalizesTimes(t *testing.T) {
	rows, err := buildOpeningHours("resource", 9, []dto.OpeningHourDTO{hours(1, "08:00", "24:00")})
	if err != nil || len(rows) != 1 {
		t.Fatalf("unexpected: %v %v", rows, err)
	}
	if rows[0].CloseTime != "24:00" || rows[0].OwnerID != 9 || rows[0].OwnerType != "resource" {
		t.Fatalf("unexpected row: %+v", rows[0])
	}
}
