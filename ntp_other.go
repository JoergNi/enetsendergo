//go:build !linux

package main

// waitForNTPSync is a no-op on non-Linux platforms (e.g. Windows test runner).
func waitForNTPSync() {}
