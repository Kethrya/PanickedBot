package commands

import (
	"testing"

	"github.com/bwmarrin/discordgo"
)

func TestParseFlexibleDate(t *testing.T) {
	est := getEasternLocation()

	tests := []struct {
		name          string
		input         string
		expectError   bool
		expectedDay   int
		expectedMonth int
		expectedYear  int
	}{
		{
			name:          "Single digit day and month (28-1-1 = Jan 1, 2028)",
			input:         "28-1-1",
			expectError:   false,
			expectedDay:   1,
			expectedMonth: 1,
			expectedYear:  2028,
		},
		{
			name:          "Single digit month, double digit day (28-1-26 = Jan 26, 2028)",
			input:         "28-1-26",
			expectError:   false,
			expectedDay:   26,
			expectedMonth: 1,
			expectedYear:  2028,
		},
		{
			name:          "Double digit month, single digit day (28-12-5 = Dec 5, 2028)",
			input:         "28-12-5",
			expectError:   false,
			expectedDay:   5,
			expectedMonth: 12,
			expectedYear:  2028,
		},
		{
			name:          "All double digits (28-01-26 = Jan 26, 2028)",
			input:         "28-01-26",
			expectError:   false,
			expectedDay:   26,
			expectedMonth: 1,
			expectedYear:  2028,
		},
		{
			name:          "End of month (25-12-31 = Dec 31, 2025)",
			input:         "25-12-31",
			expectError:   false,
			expectedDay:   31,
			expectedMonth: 12,
			expectedYear:  2025,
		},
		{
			name:          "Leap year date (24-2-29 = Feb 29, 2024)",
			input:         "24-2-29",
			expectError:   false,
			expectedDay:   29,
			expectedMonth: 2,
			expectedYear:  2024,
		},
		{
			name:          "Start of year (26-1-1 = Jan 1, 2026)",
			input:         "26-1-1",
			expectError:   false,
			expectedDay:   1,
			expectedMonth: 1,
			expectedYear:  2026,
		},
		{
			name:          "With leading/trailing whitespace (27-6-15 = Jun 15, 2027)",
			input:         "  27-6-15  ",
			expectError:   false,
			expectedDay:   15,
			expectedMonth: 6,
			expectedYear:  2027,
		},
		{
			name:        "Invalid format - missing parts",
			input:       "28-1",
			expectError: true,
		},
		{
			name:        "Invalid format - too many parts",
			input:       "28-1-26-extra",
			expectError: true,
		},
		{
			name:        "Invalid day - zero",
			input:       "28-1-0",
			expectError: true,
		},
		{
			name:        "Invalid day - too large",
			input:       "28-1-32",
			expectError: true,
		},
		{
			name:        "Invalid month - zero",
			input:       "28-0-15",
			expectError: true,
		},
		{
			name:        "Invalid month - too large",
			input:       "28-13-15",
			expectError: true,
		},
		{
			name:        "Invalid year - too large",
			input:       "100-1-15",
			expectError: true,
		},
		{
			name:        "Invalid date - Feb 30",
			input:       "28-2-30",
			expectError: true,
		},
		{
			name:        "Invalid date - Feb 29 non-leap year",
			input:       "27-2-29",
			expectError: true,
		},
		{
			name:        "Non-numeric year",
			input:       "ab-1-28",
			expectError: true,
		},
		{
			name:        "Non-numeric month",
			input:       "28-ab-15",
			expectError: true,
		},
		{
			name:        "Non-numeric day",
			input:       "28-1-ab",
			expectError: true,
		},
		{
			name:        "Empty string",
			input:       "",
			expectError: true,
		},
		{
			name:        "Wrong separator",
			input:       "28/1/15",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseFlexibleDate(tt.input, est)

			if tt.expectError {
				if err == nil {
					t.Errorf("parseFlexibleDate() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("parseFlexibleDate() unexpected error: %v", err)
				return
			}

			if result.Day() != tt.expectedDay {
				t.Errorf("parseFlexibleDate() day = %d, want %d", result.Day(), tt.expectedDay)
			}

			if int(result.Month()) != tt.expectedMonth {
				t.Errorf("parseFlexibleDate() month = %d, want %d", int(result.Month()), tt.expectedMonth)
			}

			if result.Year() != tt.expectedYear {
				t.Errorf("parseFlexibleDate() year = %d, want %d", result.Year(), tt.expectedYear)
			}

			// Check timezone
			if result.Location() != est {
				t.Errorf("parseFlexibleDate() timezone = %v, want %v", result.Location(), est)
			}
		})
	}
}


func TestNormalizeClassName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "lowercase single word",
			input:    "warrior",
			expected: "Warrior",
		},
		{
			name:     "uppercase single word",
			input:    "WARRIOR",
			expected: "Warrior",
		},
		{
			name:     "mixed case single word",
			input:    "WaRrIoR",
			expected: "Warrior",
		},
		{
			name:     "lowercase two words",
			input:    "dark knight",
			expected: "Dark Knight",
		},
		{
			name:     "uppercase two words",
			input:    "DARK KNIGHT",
			expected: "Dark Knight",
		},
		{
			name:     "mixed case two words",
			input:    "DaRk KnIgHt",
			expected: "Dark Knight",
		},
		{
			name:     "already normalized",
			input:    "Dark Knight",
			expected: "Dark Knight",
		},
		{
			name:     "extra spaces",
			input:    "dark  knight",
			expected: "Dark Knight",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeClassName(tt.input)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestValidateClassName(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedValid bool
		expectedName  string
	}{
		// Valid classes - single word
		{
			name:          "warrior lowercase",
			input:         "warrior",
			expectedValid: true,
			expectedName:  "Warrior",
		},
		{
			name:          "WARRIOR uppercase",
			input:         "WARRIOR",
			expectedValid: true,
			expectedName:  "Warrior",
		},
		{
			name:          "Ranger normalized",
			input:         "Ranger",
			expectedValid: true,
			expectedName:  "Ranger",
		},
		{
			name:          "sorceress mixed case",
			input:         "SoRcErEsS",
			expectedValid: true,
			expectedName:  "Sorceress",
		},
		// Valid classes - two words
		{
			name:          "dark knight lowercase",
			input:         "dark knight",
			expectedValid: true,
			expectedName:  "Dark Knight",
		},
		{
			name:          "DARK KNIGHT uppercase",
			input:         "DARK KNIGHT",
			expectedValid: true,
			expectedName:  "Dark Knight",
		},
		{
			name:          "Dark Knight normalized",
			input:         "Dark Knight",
			expectedValid: true,
			expectedName:  "Dark Knight",
		},
		// Invalid classes
		{
			name:          "invalid class",
			input:         "Invalid",
			expectedValid: false,
			expectedName:  "",
		},
		{
			name:          "empty string",
			input:         "",
			expectedValid: false,
			expectedName:  "",
		},
		{
			name:          "random text",
			input:         "not a class",
			expectedValid: false,
			expectedName:  "",
		},
		// Test all valid classes exist
		{
			name:          "Berserker",
			input:         "berserker",
			expectedValid: true,
			expectedName:  "Berserker",
		},
		{
			name:          "Tamer",
			input:         "tamer",
			expectedValid: true,
			expectedName:  "Tamer",
		},
		{
			name:          "Musa",
			input:         "musa",
			expectedValid: true,
			expectedName:  "Musa",
		},
		{
			name:          "Maehwa",
			input:         "maehwa",
			expectedValid: true,
			expectedName:  "Maehwa",
		},
		{
			name:          "Valkyrie",
			input:         "valkyrie",
			expectedValid: true,
			expectedName:  "Valkyrie",
		},
		{
			name:          "Kunoichi",
			input:         "kunoichi",
			expectedValid: true,
			expectedName:  "Kunoichi",
		},
		{
			name:          "Ninja",
			input:         "ninja",
			expectedValid: true,
			expectedName:  "Ninja",
		},
		{
			name:          "Wizard",
			input:         "wizard",
			expectedValid: true,
			expectedName:  "Wizard",
		},
		{
			name:          "Witch",
			input:         "witch",
			expectedValid: true,
			expectedName:  "Witch",
		},
		{
			name:          "Striker",
			input:         "striker",
			expectedValid: true,
			expectedName:  "Striker",
		},
		{
			name:          "Mystic",
			input:         "mystic",
			expectedValid: true,
			expectedName:  "Mystic",
		},
		{
			name:          "Lahn",
			input:         "lahn",
			expectedValid: true,
			expectedName:  "Lahn",
		},
		{
			name:          "Archer",
			input:         "archer",
			expectedValid: true,
			expectedName:  "Archer",
		},
		{
			name:          "Shai",
			input:         "shai",
			expectedValid: true,
			expectedName:  "Shai",
		},
		{
			name:          "Guardian",
			input:         "guardian",
			expectedValid: true,
			expectedName:  "Guardian",
		},
		{
			name:          "Hashashin",
			input:         "hashashin",
			expectedValid: true,
			expectedName:  "Hashashin",
		},
		{
			name:          "Nova",
			input:         "nova",
			expectedValid: true,
			expectedName:  "Nova",
		},
		{
			name:          "Sage",
			input:         "sage",
			expectedValid: true,
			expectedName:  "Sage",
		},
		{
			name:          "Corsair",
			input:         "corsair",
			expectedValid: true,
			expectedName:  "Corsair",
		},
		{
			name:          "Drakania",
			input:         "drakania",
			expectedValid: true,
			expectedName:  "Drakania",
		},
		{
			name:          "Woosa",
			input:         "woosa",
			expectedValid: true,
			expectedName:  "Woosa",
		},
		{
			name:          "Maegu",
			input:         "maegu",
			expectedValid: true,
			expectedName:  "Maegu",
		},
		{
			name:          "Scholar",
			input:         "scholar",
			expectedValid: true,
			expectedName:  "Scholar",
		},
		{
			name:          "Seraph",
			input:         "seraph",
			expectedValid: true,
			expectedName:  "Seraph",
		},
		{
			name:          "Wukong",
			input:         "wukong",
			expectedValid: true,
			expectedName:  "Wukong",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			normalized, valid := validateClassName(tt.input)
			if valid != tt.expectedValid {
				t.Errorf("expected valid=%v, got valid=%v", tt.expectedValid, valid)
			}
			if normalized != tt.expectedName {
				t.Errorf("expected name %q, got %q", tt.expectedName, normalized)
			}
		})
	}
}

func TestHasGuildMemberPermission(t *testing.T) {
	tests := []struct {
		name         string
		memberRoleID string
		userRoles    []string
		expected     bool
	}{
		{
			name:         "no guild member role configured",
			memberRoleID: "",
			userRoles:    []string{"role1", "role2"},
			expected:     false,
		},
		{
			name:         "user has guild member role",
			memberRoleID: "guild-role-123",
			userRoles:    []string{"role1", "guild-role-123", "role2"},
			expected:     true,
		},
		{
			name:         "user does not have guild member role",
			memberRoleID: "guild-role-123",
			userRoles:    []string{"role1", "role2"},
			expected:     false,
		},
		{
			name:         "user has no roles",
			memberRoleID: "guild-role-123",
			userRoles:    []string{},
			expected:     false,
		},
		{
			name:         "user has only guild member role",
			memberRoleID: "guild-role-123",
			userRoles:    []string{"guild-role-123"},
			expected:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &GuildConfig{
				GuildMemberRoleID: tt.memberRoleID,
			}

			i := &discordgo.InteractionCreate{
				Interaction: &discordgo.Interaction{
					Member: &discordgo.Member{
						Roles: tt.userRoles,
					},
				},
			}

			result := hasGuildMemberPermission(i, cfg)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}
