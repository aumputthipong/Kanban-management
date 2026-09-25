package service

import (
	"testing"
	"time"
)

func TestMatchesFilter(t *testing.T) {
	cases := []struct {
		filter MyWorkFilter
		group  string
		want   bool
	}{
		{"", "today", true},
		{MyWorkFilterAll, "no_date", true},
		{MyWorkFilterOverdue, "overdue", true},
		{MyWorkFilterOverdue, "today", false},
		{MyWorkFilterToday, "today", true},
		{MyWorkFilterToday, "this_week", false},
		{MyWorkFilterThisWeek, "this_week", true},
		{MyWorkFilterThisWeek, "later", false},
		{MyWorkFilterNoDate, "no_date", true},
		{MyWorkFilterNoDate, "today", false},
		{MyWorkFilter("garbage"), "today", true},
	}
	for _, c := range cases {
		if got := matchesFilter(c.filter, c.group); got != c.want {
			t.Errorf("matchesFilter(%q, %q) = %v, want %v", c.filter, c.group, got, c.want)
		}
	}
}

func TestMyWorkToday_TruncatedToBangkokMidnight(t *testing.T) {
	// 03:30 UTC = 10:30 in Bangkok, same date.
	in := time.Date(2026, 5, 28, 3, 30, 0, 0, time.UTC)
	got := MyWorkToday(in, "Asia/Bangkok")

	if got.Location().String() != "Asia/Bangkok" {
		t.Errorf("MyWorkToday location = %v, want Asia/Bangkok", got.Location())
	}
	if got.Hour() != 0 || got.Minute() != 0 || got.Second() != 0 {
		t.Errorf("MyWorkToday not truncated to midnight: %v", got)
	}
	if got.Day() != 28 || got.Month() != time.May || got.Year() != 2026 {
		t.Errorf("MyWorkToday date wrong: %v", got)
	}
}

func TestMyWorkToday_LateUTCRollsIntoNextBangkokDay(t *testing.T) {
	// 19:00 UTC = 02:00 next day in Bangkok.
	in := time.Date(2026, 5, 27, 19, 0, 0, 0, time.UTC)
	got := MyWorkToday(in, "Asia/Bangkok")
	if got.Day() != 28 {
		t.Errorf("expected Bangkok-local day=28, got %v", got)
	}
}
