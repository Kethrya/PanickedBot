package commands

import (
	"testing"
)

func TestCalculateGS(t *testing.T) {
	tests := []struct {
		name     string
		ap       *int
		aap      *int
		dp       *int
		expected int
	}{
		{
			name:     "all nil",
			ap:       nil,
			aap:      nil,
			dp:       nil,
			expected: 0,
		},
		{
			name:     "all zeros",
			ap:       intPtr(0),
			aap:      intPtr(0),
			dp:       intPtr(0),
			expected: 0,
		},
		{
			name:     "typical values",
			ap:       intPtr(300),
			aap:      intPtr(320),
			dp:       intPtr(400),
			expected: 710, // (300+320)/2 + 400 = 310 + 400 = 710
		},
		{
			name:     "only ap",
			ap:       intPtr(300),
			aap:      nil,
			dp:       nil,
			expected: 150, // 300/2 + 0 = 150
		},
		{
			name:     "only aap",
			ap:       nil,
			aap:      intPtr(320),
			dp:       nil,
			expected: 160, // 320/2 + 0 = 160
		},
		{
			name:     "only dp",
			ap:       nil,
			aap:      nil,
			dp:       intPtr(400),
			expected: 400, // 0/2 + 400 = 400
		},
		{
			name:     "ap and dp",
			ap:       intPtr(300),
			aap:      nil,
			dp:       intPtr(400),
			expected: 550, // 300/2 + 400 = 550
		},
		{
			name:     "odd ap and aap sum",
			ap:       intPtr(301),
			aap:      intPtr(320),
			dp:       intPtr(400),
			expected: 710, // (301+320)/2 + 400 = 310 + 400 = 710 (integer division)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateGS(tt.ap, tt.aap, tt.dp)
			if result != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{
			name:     "no truncation needed",
			input:    "short",
			maxLen:   10,
			expected: "short",
		},
		{
			name:     "exact length",
			input:    "exactly10!",
			maxLen:   10,
			expected: "exactly10!",
		},
		{
			name:     "truncate with ellipsis",
			input:    "This is a very long string that needs truncating",
			maxLen:   20,
			expected: "This is a very lo...",
		},
		{
			name:     "truncate to very short",
			input:    "Hello World",
			maxLen:   5,
			expected: "He...",
		},
		{
			name:     "truncate to 3 chars",
			input:    "Hello",
			maxLen:   3,
			expected: "Hel",
		},
		{
			name:     "truncate to 1 char",
			input:    "Hello",
			maxLen:   1,
			expected: "H",
		},
		{
			name:     "empty string",
			input:    "",
			maxLen:   5,
			expected: "",
		},
		{
			name:     "unicode characters no truncation",
			input:    "Hello 世界",
			maxLen:   20,
			expected: "Hello 世界",
		},
		{
			name:     "unicode characters with truncation",
			input:    "Hello 世界 こんにちは",
			maxLen:   10,
			expected: "Hello 世...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateString(tt.input, tt.maxLen)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// Helper function for tests
func intPtr(i int) *int {
	return &i
}
