package db

import (
	"strings"
	"testing"
)

func TestErrTeamAlreadyExists(t *testing.T) {
	// Test that the error constant is defined correctly
	if ErrTeamAlreadyExists == nil {
		t.Fatal("ErrTeamAlreadyExists should not be nil")
	}

	expectedMsg := "team already exists and is active"
	if ErrTeamAlreadyExists.Error() != expectedMsg {
		t.Errorf("expected error message %q, got %q", expectedMsg, ErrTeamAlreadyExists.Error())
	}
}

func TestTeamCodeGeneration(t *testing.T) {
	// While we can't test CreateTeam without a database,
	// we can test the logic of generating team codes
	tests := []struct {
		name         string
		teamName     string
		expectedCode string
	}{
		{
			name:         "single word",
			teamName:     "Alpha",
			expectedCode: "alpha",
		},
		{
			name:         "two words",
			teamName:     "Team Alpha",
			expectedCode: "team_alpha",
		},
		{
			name:         "multiple words",
			teamName:     "Red Team Alpha",
			expectedCode: "red_team_alpha",
		},
		{
			name:         "mixed case",
			teamName:     "Team BRAVO",
			expectedCode: "team_bravo",
		},
		{
			name:         "with extra spaces",
			teamName:     "Team  Alpha",
			expectedCode: "team__alpha", // Multiple spaces become multiple underscores
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Replicate the logic from CreateTeam
			code := strings.ToLower(strings.ReplaceAll(tt.teamName, " ", "_"))
			if code != tt.expectedCode {
				t.Errorf("expected code %q, got %q", tt.expectedCode, code)
			}
		})
	}
}

func TestNullStringFromPtr(t *testing.T) {
	tests := []struct {
		name          string
		input         *string
		expectedValid bool
		expectedValue string
	}{
		{
			name:          "nil pointer",
			input:         nil,
			expectedValid: false,
			expectedValue: "",
		},
		{
			name:          "empty string",
			input:         stringPtr(""),
			expectedValid: true,
			expectedValue: "",
		},
		{
			name:          "non-empty string",
			input:         stringPtr("test"),
			expectedValid: true,
			expectedValue: "test",
		},
		{
			name:          "string with spaces",
			input:         stringPtr("  test  "),
			expectedValid: true,
			expectedValue: "  test  ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := nullStringFromPtr(tt.input)
			if result.Valid != tt.expectedValid {
				t.Errorf("expected Valid=%v, got Valid=%v", tt.expectedValid, result.Valid)
			}
			if result.Valid && result.String != tt.expectedValue {
				t.Errorf("expected String=%q, got String=%q", tt.expectedValue, result.String)
			}
		})
	}
}

// Helper function for tests
func stringPtr(s string) *string {
	return &s
}
