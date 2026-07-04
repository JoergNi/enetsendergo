package main

import (
	"os"
	"strings"
	"time"
)

const (
	bootIDPath     = "/proc/sys/kernel/random/boot_id"
	syncMarkerPath = "/data/ntp_synced_boot_id"
)

// readBootID returns the kernel boot ID from path, or "" if unreadable.
func readBootID(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// clockSyncedThisBoot reports whether markerPath records an NTP sync during
// the kernel boot identified by bootID. The kernel clock is monotonic within
// a boot, so a match means the clock is still correct even if NTP is
// currently unreachable (e.g. add-on restart during an internet outage).
func clockSyncedThisBoot(bootID, markerPath string) bool {
	if bootID == "" {
		return false
	}
	data, err := os.ReadFile(markerPath)
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(data)) == bootID
}

// writeSyncMarker records bootID in markerPath after a verified NTP sync.
func writeSyncMarker(bootID, markerPath string) {
	if bootID == "" {
		return
	}
	if err := os.WriteFile(markerPath, []byte(bootID+"\n"), 0o644); err != nil {
		LogNormal("Could not write NTP sync marker: " + err.Error())
	}
}

// ntpLogInterval returns how often to log the NTP wait message after having
// waited for the given duration: every 5s in the first minute, then every
// minute, then every 5 minutes — keeps a day-long outage from flooding the log.
func ntpLogInterval(waited time.Duration) time.Duration {
	switch {
	case waited < time.Minute:
		return 5 * time.Second
	case waited < 10*time.Minute:
		return time.Minute
	default:
		return 5 * time.Minute
	}
}
