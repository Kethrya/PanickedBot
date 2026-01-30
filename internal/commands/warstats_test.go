package commands

import (
	"strings"
	"testing"
	"time"

	"PanickedBot/internal/db"
)

func TestFormatWarStatLine(t *testing.T) {
	tests := []struct {
		name     string
		stat     db.WarStats
		contains []string // strings that should be in the output
	}{
		{
			name: "no wars",
			stat: db.WarStats{
				FamilyName:    "TestFamily",
				TotalWars:     0,
				MostRecentWar: nil,
				TotalKills:    0,
				TotalDeaths:   0,
			},
			contains: []string{"TestFamily", "N/A"},
		},
		{
			name: "with wars and date",
			stat: db.WarStats{
				FamilyName:    "PlayerOne",
				TotalWars:     5,
				MostRecentWar: timePtr(time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)),
				TotalKills:    50,
				TotalDeaths:   25,
			},
			contains: []string{"PlayerOne", "5", "24-01-15", "50", "25", "2.00"},
		},
		{
			name: "zero deaths K/D",
			stat: db.WarStats{
				FamilyName:    "NoDeaths",
				TotalWars:     3,
				MostRecentWar: nil,
				TotalKills:    30,
				TotalDeaths:   0,
			},
			contains: []string{"NoDeaths", "3", "30", "0", "30.00"},
		},
		{
			name: "zero kills K/D",
			stat: db.WarStats{
				FamilyName:    "NoKills",
				TotalWars:     2,
				MostRecentWar: nil,
				TotalKills:    0,
				TotalDeaths:   10,
			},
			contains: []string{"NoKills", "2", "0", "10", "0.00"},
		},
		{
			name: "fractional K/D",
			stat: db.WarStats{
				FamilyName:    "Average",
				TotalWars:     10,
				MostRecentWar: nil,
				TotalKills:    37,
				TotalDeaths:   15,
			},
			contains: []string{"Average", "10", "37", "15", "2.47"},
		},
		{
			name: "long family name truncation",
			stat: db.WarStats{
				FamilyName:    "VeryLongFamilyNameThatExceedsTwentyCharacters",
				TotalWars:     1,
				MostRecentWar: nil,
				TotalKills:    10,
				TotalDeaths:   5,
			},
			contains: []string{"VeryLongFamilyNam...", "1", "10", "5"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatWarStatLine(tt.stat)

			// Check that all expected strings are in the output
			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("expected output to contain %q, got: %q", expected, result)
				}
			}

			// Verify output is not empty
			if result == "" {
				t.Error("expected non-empty output")
			}
		})
	}
}

// Helper function for tests
func timePtr(t time.Time) *time.Time {
	return &t
}
