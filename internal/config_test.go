package internal

import (
	"os"
	"testing"
)

func TestLoadConfigFromEnv(t *testing.T) {
	tests := []struct {
		name           string
		discordToken   string
		databaseDSN    string
		expectedError  bool
		errorContains  string
	}{
		{
			name:          "valid configuration",
			discordToken:  "test-token-123",
			databaseDSN:   "user:pass@tcp(localhost:3306)/db",
			expectedError: false,
		},
		{
			name:          "missing discord token",
			discordToken:  "",
			databaseDSN:   "user:pass@tcp(localhost:3306)/db",
			expectedError: true,
			errorContains: "DISCORD_BOT_TOKEN is not set",
		},
		{
			name:          "missing database DSN",
			discordToken:  "test-token-123",
			databaseDSN:   "",
			expectedError: true,
			errorContains: "DATABASE_DSN is not set",
		},
		{
			name:          "whitespace only discord token",
			discordToken:  "   ",
			databaseDSN:   "user:pass@tcp(localhost:3306)/db",
			expectedError: true,
			errorContains: "DISCORD_BOT_TOKEN is not set",
		},
		{
			name:          "whitespace only database DSN",
			discordToken:  "test-token-123",
			databaseDSN:   "   ",
			expectedError: true,
			errorContains: "DATABASE_DSN is not set",
		},
		{
			name:          "both missing",
			discordToken:  "",
			databaseDSN:   "",
			expectedError: true,
			errorContains: "DISCORD_BOT_TOKEN is not set",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original env vars
			origToken := os.Getenv("DISCORD_BOT_TOKEN")
			origDSN := os.Getenv("DATABASE_DSN")
			defer func() {
				os.Setenv("DISCORD_BOT_TOKEN", origToken)
				os.Setenv("DATABASE_DSN", origDSN)
			}()

			// Set test env vars
			os.Setenv("DISCORD_BOT_TOKEN", tt.discordToken)
			os.Setenv("DATABASE_DSN", tt.databaseDSN)

			cfg, err := LoadConfigFromEnv()

			if tt.expectedError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.errorContains != "" && err.Error() != tt.errorContains {
					t.Errorf("expected error containing %q, got %q", tt.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
					return
				}
				if cfg.DiscordToken != tt.discordToken {
					t.Errorf("expected DiscordToken %q, got %q", tt.discordToken, cfg.DiscordToken)
				}
				if cfg.DatabaseDSN != tt.databaseDSN {
					t.Errorf("expected DatabaseDSN %q, got %q", tt.databaseDSN, cfg.DatabaseDSN)
				}
			}
		})
	}
}
