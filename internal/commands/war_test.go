package commands

import (
	"strings"
	"testing"
)

func TestCleanCSVContent(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "CSV with markdown code blocks",
			input: "```csv\n2026-01-29\nFamilyName1,10,5\nFamilyName2,15,8\n```",
			expected: `2026-01-29
FamilyName1,10,5
FamilyName2,15,8`,
		},
		{
			name: "CSV with triple backticks only",
			input: "```\n2026-01-29\nFamilyName1,10,5\nFamilyName2,15,8\n```",
			expected: `2026-01-29
FamilyName1,10,5
FamilyName2,15,8`,
		},
		{
			name: "CSV with blank lines",
			input: `2026-01-29

FamilyName1,10,5

FamilyName2,15,8
`,
			expected: `2026-01-29
FamilyName1,10,5
FamilyName2,15,8`,
		},
		{
			name: "CSV with markdown and blank lines",
			input: "```csv\n\n2026-01-29\n\nFamilyName1,10,5\nFamilyName2,15,8\n\n```",
			expected: `2026-01-29
FamilyName1,10,5
FamilyName2,15,8`,
		},
		{
			name: "Clean CSV without any formatting",
			input: `2026-01-29
FamilyName1,10,5
FamilyName2,15,8`,
			expected: `2026-01-29
FamilyName1,10,5
FamilyName2,15,8`,
		},
		{
			name:     "Empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "Only blank lines",
			input:    "\n\n\n",
			expected: "",
		},
		{
			name:     "Only markdown markers",
			input:    "```\n```",
			expected: "",
		},
		{
			name: "CSV with leading/trailing whitespace",
			input: `  2026-01-29  
  FamilyName1,10,5  
  FamilyName2,15,8  `,
			expected: `2026-01-29
FamilyName1,10,5
FamilyName2,15,8`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cleanCSVContent(tt.input)
			if result != tt.expected {
				t.Errorf("cleanCSVContent() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestParseWarCSV(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectError   bool
		expectedDate  string
		expectedLines int
	}{
		{
			name: "Valid CSV",
			input: `2026-01-29
FamilyName1,10,5
FamilyName2,15,8`,
			expectError:   false,
			expectedDate:  "2026-01-29",
			expectedLines: 2,
		},
		{
			name: "Valid CSV with cleaned content",
			input: `2026-01-29
FamilyName1,10,5
FamilyName2,15,8`,
			expectError:   false,
			expectedDate:  "2026-01-29",
			expectedLines: 2,
		},
		{
			name: "Invalid date format",
			input: "```\nFamilyName1,10,5",
			expectError: true,
		},
		{
			name:        "Empty CSV",
			input:       "",
			expectError: true,
		},
		{
			name: "CSV with no war data",
			input: `2026-01-29
`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			warDate, warLines, err := parseWarCSV(strings.NewReader(tt.input))
			
			if tt.expectError {
				if err == nil {
					t.Errorf("parseWarCSV() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("parseWarCSV() unexpected error: %v", err)
				return
			}

			if warDate.Format("2006-01-02") != tt.expectedDate {
				t.Errorf("parseWarCSV() date = %v, want %v", warDate.Format("2006-01-02"), tt.expectedDate)
			}

			if len(warLines) != tt.expectedLines {
				t.Errorf("parseWarCSV() lines = %d, want %d", len(warLines), tt.expectedLines)
			}
		})
	}
}
