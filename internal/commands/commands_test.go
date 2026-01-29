package commands

import (
	"testing"
)

func TestFloat64Ptr(t *testing.T) {
	tests := []struct {
		name  string
		input float64
	}{
		{
			name:  "zero",
			input: 0.0,
		},
		{
			name:  "positive number",
			input: 42.5,
		},
		{
			name:  "negative number",
			input: -10.25,
		},
		{
			name:  "large number",
			input: 999999.999,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := float64Ptr(tt.input)
			if result == nil {
				t.Error("expected non-nil pointer")
				return
			}
			if *result != tt.input {
				t.Errorf("expected %f, got %f", tt.input, *result)
			}
		})
	}
}

func TestGetSpecChoices(t *testing.T) {
	choices := getSpecChoices()

	if len(choices) != 3 {
		t.Errorf("expected 3 spec choices, got %d", len(choices))
		return
	}

	expectedChoices := map[string]string{
		"Succession": "succession",
		"Awakening":  "awakening",
		"Ascension":  "ascension",
	}

	for _, choice := range choices {
		expectedValue, exists := expectedChoices[choice.Name]
		if !exists {
			t.Errorf("unexpected choice name: %s", choice.Name)
			continue
		}
		if choice.Value != expectedValue {
			t.Errorf("for choice %s, expected value %s, got %s", choice.Name, expectedValue, choice.Value)
		}
	}
}

func TestGetCommands(t *testing.T) {
	commands := GetCommands()

	if len(commands) == 0 {
		t.Error("expected at least one command")
		return
	}

	// Check that ping command exists
	foundPing := false
	for _, cmd := range commands {
		if cmd.Name == "ping" {
			foundPing = true
			if cmd.Description != "health check" {
				t.Errorf("expected ping description 'health check', got %q", cmd.Description)
			}
		}
	}

	if !foundPing {
		t.Error("expected to find 'ping' command")
	}

	// Verify command names are unique
	nameMap := make(map[string]bool)
	for _, cmd := range commands {
		if nameMap[cmd.Name] {
			t.Errorf("duplicate command name: %s", cmd.Name)
		}
		nameMap[cmd.Name] = true

		// Verify command has a name
		if cmd.Name == "" {
			t.Error("found command with empty name")
		}
	}
}
