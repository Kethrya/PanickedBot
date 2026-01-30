package commands

import (
	"fmt"
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
			input: "```csv\n26-01-29\nFamilyName1,10,5\nFamilyName2,15,8\n```",
			expected: `26-01-29
FamilyName1,10,5
FamilyName2,15,8`,
		},
		{
			name: "CSV with triple backticks only",
			input: "```\n26-01-29\nFamilyName1,10,5\nFamilyName2,15,8\n```",
			expected: `26-01-29
FamilyName1,10,5
FamilyName2,15,8`,
		},
		{
			name: "CSV with blank lines",
			input: `26-01-29

FamilyName1,10,5

FamilyName2,15,8
`,
			expected: `26-01-29
FamilyName1,10,5
FamilyName2,15,8`,
		},
		{
			name: "CSV with markdown and blank lines",
			input: "```csv\n\n26-01-29\n\nFamilyName1,10,5\nFamilyName2,15,8\n\n```",
			expected: `26-01-29
FamilyName1,10,5
FamilyName2,15,8`,
		},
		{
			name: "Clean CSV without any formatting",
			input: `26-01-29
FamilyName1,10,5
FamilyName2,15,8`,
			expected: `26-01-29
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
			input: `  26-01-29  
  FamilyName1,10,5  
  FamilyName2,15,8  `,
			expected: `26-01-29
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
			name: "Valid CSV with double-digit date (26-01-29 = Jan 29, 2026)",
			input: `26-01-29
FamilyName1,10,5
FamilyName2,15,8`,
			expectError:   false,
			expectedDate:  "26-01-29",
			expectedLines: 2,
		},
		{
			name: "Valid CSV with single-digit month (28-1-26 = Jan 26, 2028)",
			input: `28-1-26
FamilyName1,10,5
FamilyName2,15,8`,
			expectError:   false,
			expectedDate:  "28-01-26",
			expectedLines: 2,
		},
		{
			name: "Valid CSV with single-digit day (28-12-5 = Dec 5, 2028)",
			input: `28-12-5
FamilyName1,10,5
FamilyName2,15,8`,
			expectError:   false,
			expectedDate:  "28-12-05",
			expectedLines: 2,
		},
		{
			name: "Valid CSV with single-digit day and month (28-1-1 = Jan 1, 2028)",
			input: `28-1-1
FamilyName1,10,5
FamilyName2,15,8`,
			expectError:   false,
			expectedDate:  "28-01-01",
			expectedLines: 2,
		},
		{
			name: "Valid CSV with cleaned content (26-01-29 = Jan 29, 2026)",
			input: `26-01-29
FamilyName1,10,5
FamilyName2,15,8`,
			expectError:   false,
			expectedDate:  "26-01-29",
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
			input: `26-01-29
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

			// Format date as YY-MM-DD to match expected format
			actualDate := fmt.Sprintf("%02d-%02d-%02d", warDate.Year()-2000, warDate.Month(), warDate.Day())
			if actualDate != tt.expectedDate {
				t.Errorf("parseWarCSV() date = %v, want %v", actualDate, tt.expectedDate)
			}

			if len(warLines) != tt.expectedLines {
				t.Errorf("parseWarCSV() lines = %d, want %d", len(warLines), tt.expectedLines)
			}
		})
	}
}
