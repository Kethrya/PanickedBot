package commands

import (
	"github.com/bwmarrin/discordgo"

	"PanickedBot/internal"
)

// GuildConfig is a type alias for internal.GuildConfig for convenience
type GuildConfig = internal.GuildConfig

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
