package internal

import (
	"testing"
)

func TestNullIfEmpty(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected interface{}
	}{
		{
			name:     "empty string",
			input:    "",
			expected: nil,
		},
		{
			name:     "whitespace only",
			input:    "   ",
			expected: nil,
		},
		{
			name:     "tabs and spaces",
			input:    "\t  \n",
			expected: nil,
		},
		{
			name:     "non-empty string",
			input:    "test",
			expected: "test",
		},
		{
			name:     "string with spaces",
			input:    " test ",
			expected: " test ",
		},
		{
			name:     "string with leading spaces",
			input:    "  test",
			expected: "  test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NullIfEmpty(tt.input)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestNullIfEmptyPtr(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectNil   bool
		expectValue string
	}{
		{
			name:      "empty string",
			input:     "",
			expectNil: true,
		},
		{
			name:      "whitespace only",
			input:     "   ",
			expectNil: true,
		},
		{
			name:      "tabs and spaces",
			input:     "\t  \n",
			expectNil: true,
		},
		{
			name:        "non-empty string",
			input:       "test",
			expectNil:   false,
			expectValue: "test",
		},
		{
			name:        "string with spaces",
			input:       " test ",
			expectNil:   false,
			expectValue: " test ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NullIfEmptyPtr(tt.input)
			if tt.expectNil {
				if result != nil {
					t.Errorf("expected nil, got %v", *result)
				}
			} else {
				if result == nil {
					t.Errorf("expected non-nil pointer, got nil")
					return
				}
				if *result != tt.expectValue {
					t.Errorf("expected %q, got %q", tt.expectValue, *result)
				}
			}
		})
	}
}
