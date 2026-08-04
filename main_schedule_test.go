package main

import (
	"testing"
	"time"
)

func TestLivingRoomRaffstoresUpTimeUsesTenHundredBeforeVacation(t *testing.T) {
	loc := mustBerlinLocation(t)
	now := time.Date(2026, time.August, 7, 12, 0, 0, 0, loc)
	today := time.Date(2026, time.August, 7, 0, 0, 0, 0, loc)
	sunrise := time.Date(2026, time.August, 7, 6, 0, 0, 0, loc)

	got := livingRoomRaffstoresUpTime(now, today, sunrise)
	want := today.Add(10 * time.Hour)
	if !got.Equal(want) {
		t.Fatalf("expected default 10:00 local time before vacation, got %s", got.Format(time.RFC3339))
	}
}

func TestLivingRoomRaffstoresUpTimeUsesSixteenHundredOnDepartureDate(t *testing.T) {
	loc := mustBerlinLocation(t)
	now := time.Date(2026, time.August, 8, 12, 0, 0, 0, loc)
	today := time.Date(2026, time.August, 8, 0, 0, 0, 0, loc)
	sunrise := time.Date(2026, time.August, 8, 6, 1, 0, 0, loc)

	got := livingRoomRaffstoresUpTime(now, today, sunrise)
	want := today.Add(16 * time.Hour)
	if !got.Equal(want) {
		t.Fatalf("expected 16:00 local time on departure date, got %s", got.Format(time.RFC3339))
	}
}

func TestLivingRoomRaffstoresUpTimeKeepsSixteenHundredOnReturnDate(t *testing.T) {
	loc := mustBerlinLocation(t)
	now := time.Date(2026, time.August, 22, 12, 0, 0, 0, loc)
	today := time.Date(2026, time.August, 22, 0, 0, 0, 0, loc)
	sunrise := time.Date(2026, time.August, 22, 6, 15, 0, 0, loc)

	got := livingRoomRaffstoresUpTime(now, today, sunrise)
	want := today.Add(16 * time.Hour)
	if !got.Equal(want) {
		t.Fatalf("expected 16:00 local time on return date, got %s", got.Format(time.RFC3339))
	}
}

func TestLivingRoomRaffstoresUpTimeRevertsAfterVacation(t *testing.T) {
	loc := mustBerlinLocation(t)
	now := time.Date(2026, time.August, 23, 12, 0, 0, 0, loc)
	today := time.Date(2026, time.August, 23, 0, 0, 0, 0, loc)
	sunrise := time.Date(2026, time.August, 23, 6, 16, 0, 0, loc)

	got := livingRoomRaffstoresUpTime(now, today, sunrise)
	want := today.Add(10 * time.Hour)
	if !got.Equal(want) {
		t.Fatalf("expected default 10:00 local time after vacation, got %s", got.Format(time.RFC3339))
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