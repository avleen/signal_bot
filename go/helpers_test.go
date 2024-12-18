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

func TestGetMessageRoot(t *testing.T) {
	// Define the test data
	testData := `{
        "envelope": {
            "source": "+1234567890",
            "sourceNumber": "+1234567890",
            "sourceUuid": "<fake_uuid>",
            "sourceName": "Test User",
            "sourceDevice": 1,
            "timestamp": 1733066028521,
            "dataMessage": {
                "timestamp": 1733066028521,
                "message": "Testing messages",
                "expiresInSeconds": 0,
                "viewOnce": false,
                "attachments": [
                    {
                        "contentType": "image/jpeg",
                        "filename": "galaxy.jpg",
                        "id": "r4aFDRWmi_z2dfVh5iqC.jpg",
                        "size": 273635,
                        "width": 2048,
                        "height": 2048,
                        "caption": null,
                        "uploadTimestamp": null
                    }
                ],
                "groupInfo": {
                    "groupId": "VGVzdA==",
                    "type": "DELIVER"
                }
            }
        },
        "account": "+1234567890"
    }`

	// Call getMessageRoot with the test data
	container, msgStruct, err := getMessageRoot(testData)
	if err != nil {
		t.Fatalf("getMessageRoot returned an error: %v", err)
	}

	// Check the container
	if container == nil {
		t.Fatalf("Expected container to be non-nil")
	}

	// Check the msgStruct
	if msgStruct == nil {
		t.Fatalf("Expected msgStruct to be non-nil")
	}

	// Check specific fields in msgStruct
	expectedMessage := "Testing messages"
	if msgStruct["message"] != expectedMessage {
		t.Errorf("Expected message %s, got %s", expectedMessage, msgStruct["message"])
	}

	expectedGroupId := "VGVzdA=="
	groupInfo, ok := msgStruct["groupInfo"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected groupInfo to be of type map[string]interface{}")
	}
	if groupInfo["groupId"] != expectedGroupId {
		t.Errorf("Expected groupId %s, got %s", expectedGroupId, groupInfo["groupId"])
	}
}
