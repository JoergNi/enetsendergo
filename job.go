package main

import (
	"fmt"
	"sync"
	"time"
)

// Job represents a scheduled daily task.
type Job struct {
	Name           string
	Time           time.Time
	Action         func()
	DoneForToday   bool
	IgnoreOnWeekends bool
}

func NewJob(name string, t time.Time, action func(), ignoreOnWeekends bool) *Job {
	j := &Job{
		Name:             name,
		Time:             t,
		Action:           action,
		IgnoreOnWeekends: ignoreOnWeekends,
	}
	now := time.Now()
	if t.Before(now) {
		LogNormal(fmt.Sprintf("[Job] %s already past (%s, now=%s), skipping", name, t.Format("15:04:05"), now.Format("15:04:05")))
		j.DoneForToday = true
	}
	return j
}

func (j *Job) Check() {
	if j.DoneForToday {
		return
	}
	if j.IgnoreOnWeekends {
		wd := time.Now().Weekday()
		if wd == time.Saturday || wd == time.Sunday {
			j.DoneForToday = true
			return
		}
	}
	if time.Now().After(j.Time) {
		LogNormal(fmt.Sprintf("[Job] %s firing (scheduled %s)", j.Name, j.Time.Format("15:04:05")))
		j.Action()
		j.DoneForToday = true
	}
}

// Logging
var (
	normalLog     []string
	normalLogMu   sync.Mutex
	debugLog      []string
	debugLogMu    sync.Mutex
	debugLogMax   = 10000
)

func LogNormal(message string) {
	line := fmt.Sprintf("%s %s", time.Now().Format("2006-01-02 15:04:05"), message)
	fmt.Println(line)
	normalLogMu.Lock()
	normalLog = append(normalLog, line)
	normalLogMu.Unlock()
}

func LogDebug(message string) {
	line := fmt.Sprintf("%s %s", time.Now().Format("2006-01-02 15:04:05"), message)
	fmt.Println(line)
	debugLogMu.Lock()
	if len(debugLog) >= debugLogMax {
		debugLog = debugLog[1:]
	}
	debugLog = append(debugLog, line)
	debugLogMu.Unlock()
}

func GetNormalLog() []string {
	normalLogMu.Lock()
	defer normalLogMu.Unlock()
	cp := make([]string, len(normalLog))
	copy(cp, normalLog)
	return cp
}

func GetDebugLog() []string {
	debugLogMu.Lock()
	defer debugLogMu.Unlock()
	cp := make([]string, len(debugLog))
	copy(cp, debugLog)
	return cp
}
