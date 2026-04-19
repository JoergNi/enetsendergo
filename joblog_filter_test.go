package main

import (
	"testing"
	"time"
)

func TestFilterJobLog_ExcludesEntriesBeforeCutoff(t *testing.T) {
	entries := []string{
		"2026-04-01 10:00:00 [Job] old entry",
		"2026-04-10 10:00:00 [Job] boundary entry",
		"2026-04-13 20:00:00 [Job] recent entry",
	}

	result := FilterJobLog(entries, "2026-04-10")

	if len(result) != 2 {
		t.Fatalf("expected 2, got %d", len(result))
	}
	for _, r := range result {
		assertNotContains(t, r, "old entry")
	}
}

func TestFilterJobLog_IncludesEntriesOnCutoffDate(t *testing.T) {
	entries := []string{
		"2026-04-10 00:00:01 [Job] early",
		"2026-04-10 23:59:59 [Job] late",
	}

	result := FilterJobLog(entries, "2026-04-10")

	if len(result) != 2 {
		t.Fatalf("expected 2, got %d", len(result))
	}
}

func TestFilterJobLog_AllEntriesOld_ReturnsEmpty(t *testing.T) {
	entries := []string{"2020-01-01 00:00:00 [Job] ancient"}

	result := FilterJobLog(entries, time.Now().AddDate(0, 0, -10).Format("2006-01-02"))

	if len(result) != 0 {
		t.Fatalf("expected 0, got %d", len(result))
	}
}

func TestFilterJobLog_EmptyInput_ReturnsEmpty(t *testing.T) {
	result := FilterJobLog([]string{}, time.Now().Format("2006-01-02"))

	if len(result) != 0 {
		t.Fatalf("expected 0, got %d", len(result))
	}
}

func TestFilterJobLog_AllEntriesRecent_ReturnsAll(t *testing.T) {
	today := time.Now().Format("2006-01-02")
	entries := []string{
		today + " 08:00:00 [Job] morning",
		today + " 20:00:00 [Job] evening",
	}

	result := FilterJobLog(entries, time.Now().AddDate(0, 0, -10).Format("2006-01-02"))

	if len(result) != 2 {
		t.Fatalf("expected 2, got %d", len(result))
	}
}

func assertNotContains(t *testing.T, haystack, needle string) {
	t.Helper()
	for i := 0; i <= len(haystack)-len(needle); i++ {
		if haystack[i:i+len(needle)] == needle {
			t.Errorf("expected %q to NOT contain %q", haystack, needle)
			return
		}
	}
}
