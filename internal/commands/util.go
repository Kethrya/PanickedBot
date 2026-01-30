package commands

import (
	"database/sql"
	"log"
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
