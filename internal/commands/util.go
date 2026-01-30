package commands

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"

	"PanickedBot/internal"
	"PanickedBot/internal/db"
)

// GuildConfig is a type alias for internal.GuildConfig for convenience
type GuildConfig = internal.GuildConfig

// getEasternLocation returns the Eastern timezone location (America/New_York)
// This is a convenience wrapper around the internal.GetEasternLocation function
func getEasternLocation() *time.Location {
	return internal.GetEasternLocation()
}

// parseFlexibleDate parses a date string in YY-MM-DD format where month and day
// can be either single or double digits (e.g., "28-1-26", "28-01-26", "28-1-1", "28-01-01")
// Returns the parsed time in the specified timezone location
func parseFlexibleDate(dateStr string, loc *time.Location) (time.Time, error) {
	// Split the date string by dashes
	parts := strings.Split(strings.TrimSpace(dateStr), "-")
	if len(parts) != 3 {
		return time.Time{}, fmt.Errorf("invalid date format: expected YY-MM-DD (e.g., 28-1-26 or 28-01-26)")
	}

	// Parse year, month, and day (in YY-MM-DD order)
	year, err := strconv.Atoi(parts[0])
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid year value: %s", parts[0])
	}

	month, err := strconv.Atoi(parts[1])
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid month value: %s", parts[1])
	}

	day, err := strconv.Atoi(parts[2])
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid day value: %s", parts[2])
	}

	// Validate ranges
	if year < 0 || year > 99 {
		return time.Time{}, fmt.Errorf("year must be between 0 and 99, got %d", year)
	}
	if month < 1 || month > 12 {
		return time.Time{}, fmt.Errorf("month must be between 1 and 12, got %d", month)
	}
	if day < 1 || day > 31 {
		return time.Time{}, fmt.Errorf("day must be between 1 and 31, got %d", day)
	}

	// Convert 2-digit year to 4-digit year (assuming 2000s)
	fullYear := 2000 + year

	// Create the date in the specified location
	date := time.Date(fullYear, time.Month(month), day, 0, 0, 0, 0, loc)

	// Validate that the date is valid (e.g., not Feb 30)
	if date.Day() != day || int(date.Month()) != month {
		return time.Time{}, fmt.Errorf("invalid date: %d-%d-%d does not exist", year, month, day)
	}

	return date, nil
}

// validClasses is the list of valid Black Desert Online classes
var validClasses = map[string]bool{
	"Warrior":     true,
	"Ranger":      true,
	"Sorceress":   true,
	"Berserker":   true,
	"Tamer":       true,
	"Musa":        true,
	"Maehwa":      true,
	"Valkyrie":    true,
	"Kunoichi":    true,
	"Ninja":       true,
	"Wizard":      true,
	"Witch":       true,
	"Dark Knight": true,
	"Striker":     true,
	"Mystic":      true,
	"Lahn":        true,
	"Archer":      true,
	"Shai":        true,
	"Guardian":    true,
	"Hashashin":   true,
	"Nova":        true,
	"Sage":        true,
	"Corsair":     true,
	"Drakania":    true,
	"Woosa":       true,
	"Maegu":       true,
	"Scholar":     true,
	"Seraph":      true,
	"Wukong":      true,
}

// normalizeClassName converts a class name to the correct capitalization format
// Only the first letter of each word should be capitalized
func normalizeClassName(className string) string {
	// Split by spaces for multi-word classes like "Dark Knight"
	words := strings.Fields(className)
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(string(word[0])) + strings.ToLower(word[1:])
		}
	}
	return strings.Join(words, " ")
}

// validateClassName checks if a class name is valid and returns the normalized name
func validateClassName(className string) (string, bool) {
	normalized := normalizeClassName(className)
	if validClasses[normalized] {
		return normalized, true
	}
	return "", false
}

// hasOfficerPermission checks if user has officer role or admin permissions
func hasOfficerPermission(s *discordgo.Session, i *discordgo.InteractionCreate, cfg *GuildConfig) bool {
	// Check admin permission first
	perms, err := s.UserChannelPermissions(i.Member.User.ID, i.ChannelID)
	if err == nil && ((perms&discordgo.PermissionManageGuild) != 0 || (perms&discordgo.PermissionAdministrator) != 0) {
		return true
	}

	// Check officer role if configured
	if cfg.OfficerRoleID != "" {
		for _, roleID := range i.Member.Roles {
			if roleID == cfg.OfficerRoleID {
				return true
			}
		}
	}

	return false
}

// hasGuildMemberPermission checks if user has guild member role
func hasGuildMemberPermission(i *discordgo.InteractionCreate, cfg *GuildConfig) bool {
	if cfg.GuildMemberRoleID == "" {
		return false
	}

	for _, roleID := range i.Member.Roles {
		if roleID == cfg.GuildMemberRoleID {
			return true
		}
	}

	return false
}

// getOrCreateMember retrieves a member by Discord user ID, creating a new one if it doesn't exist.
// This is a helper to reduce code duplication across command handlers.
// Returns the member and any error encountered.
func getOrCreateMember(dbx *db.DB, guildID, userID, username, contextName string) (*internal.Member, error) {
	m, err := internal.GetMemberByDiscordUserID(dbx, guildID, userID)
	if err == sql.ErrNoRows {
		// Create new member - use Discord username as default family name
		memberID, err := internal.CreateMember(dbx, guildID, userID, username)
		if err != nil {
			log.Printf("%s create error: %v", contextName, err)
			return nil, err
		}

		// Get the newly created member
		m, err = internal.GetMemberByDiscordUserID(dbx, guildID, userID)
		if err != nil {
			log.Printf("%s lookup after create error: %v", contextName, err)
			return nil, err
		}

		log.Printf("Created new member ID %d for user %s", memberID, username)
		return m, nil
	} else if err != nil {
		log.Printf("%s lookup error: %v", contextName, err)
		return nil, err
	}

	return m, nil
}
