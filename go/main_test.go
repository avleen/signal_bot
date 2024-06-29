package main

import (
	"testing"
	"time"
)

func TestCalculateStarttimeAndCount(t *testing.T) {
	c := TimeCountCalculator{
		StartTime: int(time.Now().Add(-time.Duration(24)*time.Hour).Unix() * 1000),
		Count:     0,
	}
	// Test case 1: No arguments
	starttime, count, err := c.calculateStarttimeAndCount([]string{})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if starttime != c.StartTime || count != c.Count {
		t.Errorf("expected starttime and count to be 0, got %d and %d", starttime, count)
	}

	// Test case 2: Invalid argument
	_, _, err = c.calculateStarttimeAndCount([]string{"!summary", "abc"})
	if err == nil {
		t.Error("expected error, got nil")
	}

	// Test case 3: Hours argument
	starttime, count, err = c.calculateStarttimeAndCount([]string{"!summary", "24h"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	expectedStarttime := int(time.Now().Add(-time.Duration(24)*time.Hour).Unix() * 1000)
	if starttime != expectedStarttime || count != 0 {
		t.Errorf("expected starttime %d and count 0, got %d and %d", expectedStarttime, starttime, count)
	}

	// Test case 4: Count argument
	starttime, count, err = c.calculateStarttimeAndCount([]string{"!summary", "10"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if starttime != 0 || count != 10 {
		t.Errorf("expected starttime 0 and count 10, got %d and %d", starttime, count)
	}
}
