package main

import (
	"database/sql"
	"os"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestEncodeGroupIdToBase64(t *testing.T) {
	groupId := "exampleGroupId"
	expectedResult := "group.ZXhhbXBsZUdyb3VwSWQ="
	result := encodeGroupIdToBase64(groupId)
	if result != expectedResult {
		t.Errorf("expected result %s, got %s", expectedResult, result)
	}
}
func TestCompileLogs(t *testing.T) {
	// Set up a test sqlite database
	os.Create("test_db.db")
	defer os.Remove("test_db.db")
	db, err := sql.Open("sqlite3", "test_db.db")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer db.Close()
	_, err = db.Exec("CREATE TABLE test (log TEXT)")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, err = db.Exec("INSERT INTO test (log) VALUES ('Log 1'), ('Log 2'), ('Log 3')")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Select the rows from the test database
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer db.Close()
	rows, err := db.Query("SELECT log FROM test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer rows.Close()

	expectedResult := "Log 1\nLog 2\nLog 3\n"
	result, err := compileLogs(rows)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if result != expectedResult {
		t.Errorf("expected result %q, got %q", expectedResult, result)
	}
}
func TestGetNumberFromString(t *testing.T) {
	// Test success case
	input := "12h"
	expectedResult := 12
	result, err := getNumberFromString(input)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result != expectedResult {
		t.Errorf("expected result %d, got %d", expectedResult, result)
	}

	// Test success case
	input = "12"
	_, err = getNumberFromString(input)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result != expectedResult {
		t.Errorf("expected result %d, got %d", expectedResult, result)
	}

	// Test failure case
	input = "h"
	_, err = getNumberFromString(input)
	if err == nil {
		t.Errorf("expected error, got nil")
	}

	// Test failure case
	input = ""
	_, err = getNumberFromString(input)
	if err == nil {
		t.Errorf("expected error, got nil")
	}
}
