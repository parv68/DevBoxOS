package orchestrator

import (
	"reflect"
	"testing"
)

func TestParseCommand_Empty(t *testing.T) {
	result := parseCommand("")
	if result != nil {
		t.Errorf("expected nil for empty command, got %v", result)
	}
}

func TestParseCommand_Simple(t *testing.T) {
	result := parseCommand("npm start")
	expected := []string{"sh", "-c", "npm start"}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("parseCommand() = %v, want %v", result, expected)
	}
}

func TestParseCommand_WithArgs(t *testing.T) {
	result := parseCommand("node server.js --port 3000")
	expected := []string{"sh", "-c", "node server.js --port 3000"}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("parseCommand() = %v, want %v", result, expected)
	}
}

func TestParseCommand_MultiWord(t *testing.T) {
	result := parseCommand("echo hello world && sleep 10")
	expected := []string{"sh", "-c", "echo hello world && sleep 10"}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("parseCommand() = %v, want %v", result, expected)
	}
}
