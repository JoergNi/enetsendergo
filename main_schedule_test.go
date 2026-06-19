package main

import (
	"testing"
	"time"
)

func TestLivingRoomRaffstoresUpTimeUsesSixteenHundredBeforeCutoff(t *testing.T) {
	loc := mustBerlinLocation(t)
	now := time.Date(2026, time.June, 15, 12, 0, 0, 0, loc)
	today := time.Date(2026, time.June, 15, 0, 0, 0, 0, loc)
	sunrise := time.Date(2026, time.June, 15, 5, 12, 0, 0, loc)

	got := livingRoomRaffstoresUpTime(now, today, sunrise)
	want := today.Add(16 * time.Hour)
	if !got.Equal(want) {
		t.Fatalf("expected 16:00 local time before cutoff, got %s", got.Format(time.RFC3339))
	}
}

func TestLivingRoomRaffstoresUpTimeKeepsSixteenHundredOnCutoffDate(t *testing.T) {
	loc := mustBerlinLocation(t)
	now := time.Date(2026, time.June, 22, 12, 0, 0, 0, loc)
	today := time.Date(2026, time.June, 22, 0, 0, 0, 0, loc)
	sunrise := time.Date(2026, time.June, 22, 5, 9, 0, 0, loc)

	got := livingRoomRaffstoresUpTime(now, today, sunrise)
	want := today.Add(16 * time.Hour)
	if !got.Equal(want) {
		t.Fatalf("expected 16:00 local time on cutoff date, got %s", got.Format(time.RFC3339))
	}
}

func TestLivingRoomRaffstoresUpTimeRevertsAfterCutoff(t *testing.T) {
	loc := mustBerlinLocation(t)
	now := time.Date(2026, time.June, 23, 12, 0, 0, 0, loc)
	today := time.Date(2026, time.June, 23, 0, 0, 0, 0, loc)
	sunrise := time.Date(2026, time.June, 23, 5, 8, 0, 0, loc)

	got := livingRoomRaffstoresUpTime(now, today, sunrise)
	want := today.Add(10 * time.Hour)
	if !got.Equal(want) {
		t.Fatalf("expected default 10:00 local time after cutoff, got %s", got.Format(time.RFC3339))
	}
}

func mustBerlinLocation(t *testing.T) *time.Location {
	t.Helper()

	loc, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		t.Fatalf("load Europe/Berlin location: %v", err)
	}
	return loc
}