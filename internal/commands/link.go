package commands

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"

	"PanickedBot/internal"
	"PanickedBot/internal/db"
	"PanickedBot/internal/discord"
)

func handleLink(s *discordgo.Session, i *discordgo.InteractionCreate, dbx *db.DB, cfg *GuildConfig) {
	if !hasOfficerPermission(s, i, cfg) {
		discord.RespondEphemeral(s, i, "You need officer role or admin permission to use this command.")
		return
	}

	// Parse options
	var targetUser *discordgo.User
	var familyName string

	for _, opt := range i.ApplicationCommandData().Options {
		switch opt.Name {
		case "member":
			targetUser = opt.UserValue(s)
		case "family_name":
			familyName = opt.StringValue()
		}
	}

	if targetUser == nil {
		discord.RespondEphemeral(s, i, "Member is required.")
		return
	}

	if familyName == "" {
		discord.RespondEphemeral(s, i, "Family name is required.")
		return
	}

	// Get display name from Discord
	displayName := getDiscordDisplayName(s, i.GuildID, targetUser.ID)

	// Try to get existing member by Discord user ID (including inactive)
	m, err := internal.GetMemberByDiscordUserIDIncludingInactive(dbx, i.GuildID, targetUser.ID)
	if err == sql.ErrNoRows {
		// Member doesn't exist, create new one with the provided family name
		memberID, err := internal.CreateMember(dbx, i.GuildID, targetUser.ID, familyName)
		if err != nil {
			if isDuplicateFamilyNameError(err) {
				discord.RespondEphemeral(s, i, fmt.Sprintf("Family name '%s' is already in use by another member.", familyName))
				return
			}
			log.Printf("link create error: %v", err)
			discord.RespondEphemeral(s, i, "Failed to link member. Please try again.")
			return
		}

		// Update display name for the newly created member
		fields := internal.UpdateFields{
			DisplayName: &displayName,
		}
		err = internal.UpdateMember(dbx, memberID, fields)
		if err != nil {
			log.Printf("link update display name error: %v", err)
			// Non-fatal, continue
		}

		discord.RespondText(s, i, fmt.Sprintf("Successfully linked %s to family name '%s'.", targetUser.Mention(), familyName))
		return
	} else if err != nil {
		log.Printf("link lookup error: %v", err)
		discord.RespondEphemeral(s, i, "Failed to link member. Please try again.")
		return
	}

	// Member exists, update family name and display name
	fields := internal.UpdateFields{
		FamilyName:  &familyName,
		DisplayName: &displayName,
	}

	err = internal.UpdateMember(dbx, m.ID, fields)
	if err != nil {
		if isDuplicateFamilyNameError(err) {
			discord.RespondEphemeral(s, i, fmt.Sprintf("Family name '%s' is already in use by another member.", familyName))
			return
		}
		log.Printf("link update error: %v", err)
		discord.RespondEphemeral(s, i, "Failed to link member. Please try again.")
		return
	}

	discord.RespondText(s, i, fmt.Sprintf("Successfully updated %s's family name to '%s'.", targetUser.Mention(), familyName))
}
