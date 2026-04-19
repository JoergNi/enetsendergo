package main

import (
	"testing"
	"time"
)

func TestJob_FutureTime_NotFiredOnCheck(t *testing.T) {
	fired := false
	job := NewJob("test", time.Now().Add(1*time.Hour), func() { fired = true }, false)

	job.Check()

	if fired {
		t.Error("job should not have fired")
	}
	if job.DoneForToday {
		t.Error("job should not be marked done")
	}
}

func TestJob_PastTimeAtCreation_MarkedDoneWithoutFiring(t *testing.T) {
	fired := false
	job := NewJob("test", time.Now().Add(-60*time.Second), func() { fired = true }, false)

	if !job.DoneForToday {
		t.Error("job should be marked done at creation for past time")
	}
	if fired {
		t.Error("constructor must not call the action")
	}
}

func TestJob_Check_FiresWhenTimePassed(t *testing.T) {
	fired := false
	job := NewJob("test", time.Now().Add(1*time.Hour), func() { fired = true }, false)
	job.Time = time.Now().Add(-1 * time.Second)

	job.Check()

	if !fired {
		t.Error("job should have fired")
	}
	if !job.DoneForToday {
		t.Error("job should be marked done")
	}
}

func TestJob_Check_DoesNotFireTwice(t *testing.T) {
	count := 0
	job := NewJob("test", time.Now().Add(1*time.Hour), func() { count++ }, false)
	job.Time = time.Now().Add(-1 * time.Second)

	job.Check()
	job.Check()

	if count != 1 {
		t.Errorf("expected 1 fire, got %d", count)
	}
}

func TestJob_Name_IsSet(t *testing.T) {
	job := NewJob("MyBlind down", time.Now().Add(1*time.Hour), func() {}, false)

	if job.Name != "MyBlind down" {
		t.Errorf("expected name 'MyBlind down', got '%s'", job.Name)
	}
}
