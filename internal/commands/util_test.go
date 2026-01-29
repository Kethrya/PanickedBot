package commands

import (
	"testing"

	"github.com/bwmarrin/discordgo"
)

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
