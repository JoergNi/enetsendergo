package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestReadBootID(t *testing.T) {
	dir := t.TempDir()

	if got := readBootID(filepath.Join(dir, "missing")); got != "" {
		t.Errorf("missing file: got %q, want empty", got)
	}

	p := filepath.Join(dir, "boot_id")
	if err := os.WriteFile(p, []byte("abc-123\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := readBootID(p); got != "abc-123" {
		t.Errorf("got %q, want abc-123 (trimmed)", got)
	}
}

func TestClockSyncedThisBoot(t *testing.T) {
	dir := t.TempDir()
	marker := filepath.Join(dir, "marker")

	if clockSyncedThisBoot("abc-123", marker) {
		t.Error("missing marker: want false")
	}
	if clockSyncedThisBoot("", marker) {
		t.Error("empty boot ID: want false")
	}

	writeSyncMarker("abc-123", marker)
	if !clockSyncedThisBoot("abc-123", marker) {
		t.Error("matching marker: want true")
	}
	if clockSyncedThisBoot("other-boot", marker) {
		t.Error("mismatched boot ID: want false")
	}
}

func TestWriteSyncMarkerEmptyBootID(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "marker")
	writeSyncMarker("", marker)
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Error("empty boot ID must not create a marker file")
	}
}

func TestNtpLogInterval(t *testing.T) {
	cases := []struct {
		waited time.Duration
		want   time.Duration
	}{
		{0, 5 * time.Second},
		{59 * time.Second, 5 * time.Second},
		{time.Minute, time.Minute},
		{9 * time.Minute, time.Minute},
		{10 * time.Minute, 5 * time.Minute},
		{24 * time.Hour, 5 * time.Minute},
	}
	for _, c := range cases {
		if got := ntpLogInterval(c.waited); got != c.want {
			t.Errorf("ntpLogInterval(%v) = %v, want %v", c.waited, got, c.want)
		}
	}
}
