package appmanager

import (
	"testing"
	"time"

	"tg-monitor-bot/internal/storage"
)

func mustLoad(t *testing.T, name string) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation(name)
	if err != nil {
		t.Fatalf("LoadLocation(%q): %v", name, err)
	}
	return loc
}

func TestComputeDaysLeft(t *testing.T) {
	utc := mustLoad(t, "UTC")

	tests := []struct {
		name   string
		target string
		now    time.Time
		want   int
	}{
		{"same day", "2026-06-23", time.Date(2026, 6, 23, 8, 0, 0, 0, utc), 0},
		{"one day", "2026-06-24", time.Date(2026, 6, 23, 23, 59, 0, 0, utc), 1},
		{"ten days", "2026-07-03", time.Date(2026, 6, 23, 0, 0, 0, 0, utc), 10},
		{"past date", "2026-06-20", time.Date(2026, 6, 23, 12, 0, 0, 0, utc), -3},
		{"year span", "2027-01-01", time.Date(2026, 1, 1, 0, 0, 0, 0, utc), 365},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := computeDaysLeft(tt.target, utc, tt.now)
			if err != nil {
				t.Fatalf("computeDaysLeft: %v", err)
			}
			if got != tt.want {
				t.Errorf("computeDaysLeft(%q, now=%v) = %d, want %d", tt.target, tt.now, got, tt.want)
			}
		})
	}
}

func TestComputeDaysLeftAcrossDST(t *testing.T) {
	// Kyiv switches to DST on 2026-03-29. A day count spanning the transition
	// must still be whole days thanks to rounding.
	loc := mustLoad(t, "Europe/Kyiv")
	now := time.Date(2026, 3, 28, 10, 0, 0, 0, loc)
	got, err := computeDaysLeft("2026-03-30", loc, now)
	if err != nil {
		t.Fatalf("computeDaysLeft: %v", err)
	}
	if got != 2 {
		t.Errorf("days across DST = %d, want 2", got)
	}
}

func TestCountdownDue(t *testing.T) {
	utc := mustLoad(t, "UTC")
	base := &storage.Countdown{NotifyTime: "09:00", Timezone: "UTC"}

	t.Run("before notify time", func(t *testing.T) {
		c := *base
		now := time.Date(2026, 6, 23, 8, 30, 0, 0, utc)
		if countdownDue(&c, utc, now) {
			t.Error("should not be due before notify time")
		}
	})

	t.Run("after notify time, not sent", func(t *testing.T) {
		c := *base
		now := time.Date(2026, 6, 23, 9, 30, 0, 0, utc)
		if !countdownDue(&c, utc, now) {
			t.Error("should be due after notify time when not yet sent")
		}
	})

	t.Run("already sent today", func(t *testing.T) {
		c := *base
		c.LastSentDate = "2026-06-23"
		now := time.Date(2026, 6, 23, 9, 30, 0, 0, utc)
		if countdownDue(&c, utc, now) {
			t.Error("should not be due when already sent today")
		}
	})

	t.Run("sent yesterday, due again", func(t *testing.T) {
		c := *base
		c.LastSentDate = "2026-06-22"
		now := time.Date(2026, 6, 23, 9, 0, 0, 0, utc)
		if !countdownDue(&c, utc, now) {
			t.Error("should be due the next day at notify time")
		}
	})
}

func TestRenderCountdownMessage(t *testing.T) {
	tests := []struct {
		template string
		days     int
		want     string
	}{
		{"{} days left", 5, "5 days left"},
		{"Only {} days until launch ({} !)", 3, "Only 3 days until launch (3 !)"},
		{"no placeholder", 7, "no placeholder"},
		{"{} day(s)", -2, "-2 day(s)"},
	}
	for _, tt := range tests {
		if got := renderCountdownMessage(tt.template, tt.days); got != tt.want {
			t.Errorf("renderCountdownMessage(%q, %d) = %q, want %q", tt.template, tt.days, got, tt.want)
		}
	}
}

func TestInitialLastSentDate(t *testing.T) {
	utc := mustLoad(t, "UTC")
	c := &storage.Countdown{NotifyTime: "09:00", Timezone: "UTC"}

	// Created before notify time -> should still fire today (empty guard)
	if got := initialLastSentDate(c, utc, time.Date(2026, 6, 23, 8, 0, 0, 0, utc)); got != "" {
		t.Errorf("before notify time = %q, want empty", got)
	}

	// Created after notify time -> guard set to today so it won't fire today
	if got := initialLastSentDate(c, utc, time.Date(2026, 6, 23, 10, 0, 0, 0, utc)); got != "2026-06-23" {
		t.Errorf("after notify time = %q, want 2026-06-23", got)
	}
}
