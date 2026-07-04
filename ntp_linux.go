//go:build linux

package main

import (
	"fmt"
	"syscall"
	"time"
)

// waitForNTPSync blocks until the kernel reports the clock is synchronized
// (STA_UNSYNC cleared by the NTP daemon). On a Pi without a hardware RTC the
// clock starts from fake-hwclock and NTP may step it forward or backward during
// startup. We must not call initialize() until the clock is correct or we will
// incorrectly mark future jobs as "already past".
//
// Exception: if the marker file records an NTP sync during this same kernel
// boot, the clock has been ticking monotonically since that sync and is
// trusted immediately — an add-on restart during an internet outage must not
// stall the scheduler when the Pi itself never rebooted.
func waitForNTPSync() {
	const staUnsync = 0x0040 // STA_UNSYNC: set while clock is not yet synchronized
	if clockSyncedThisBoot(readBootID(bootIDPath), syncMarkerPath) {
		LogNormal("Clock already NTP-synced this kernel boot (marker) — starting scheduler without waiting")
		return
	}
	start := time.Now()
	attempt := 0
	var lastLog time.Time
	for {
		var tx syscall.Timex
		syscall.Adjtimex(&tx)
		if tx.Status&staUnsync == 0 {
			// Verify the clock is stable: sleep 1s and check for a large step.
			// NTP may clear STA_UNSYNC just before applying a backward step
			// (fake-hwclock ahead), causing initialize() to see a wrong time.
			t1 := time.Now()
			time.Sleep(1 * time.Second)
			t2 := time.Now()
			elapsed := t2.Sub(t1)
			if elapsed < 500*time.Millisecond || elapsed > 2*time.Second {
				LogNormal(fmt.Sprintf("Clock jumped after NTP sync (elapsed=%v) — re-checking", elapsed))
				attempt++
				continue
			}
			if attempt > 0 {
				LogNormal(fmt.Sprintf("NTP synchronized after %s — time is %s",
					time.Since(start).Truncate(time.Second), t2.Format("15:04:05")))
			}
			writeSyncMarker(readBootID(bootIDPath), syncMarkerPath)
			return
		}
		attempt++
		now := time.Now()
		if lastLog.IsZero() || now.Sub(lastLog) >= ntpLogInterval(now.Sub(start)) {
			LogNormal(fmt.Sprintf("Waiting for NTP sync (%s elapsed, attempt %d)…",
				now.Sub(start).Truncate(time.Second), attempt))
			lastLog = now
		}
		time.Sleep(5 * time.Second)
	}
}
