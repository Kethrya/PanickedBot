package commands

import (
	"database/sql"
	"errors"
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"

	"PanickedBot/internal"
	"PanickedBot/internal/db"
	"PanickedBot/internal/discord"
)

func handleMerc(s *discordgo.Session, i *discordgo.InteractionCreate, dbx *db.DB, cfg *GuildConfig) {
	if !hasOfficerPermission(s, i, cfg) {
		discord.RespondEphemeral(s, i, "You need officer role or admin permission to use this command.")
		return
	}

	options := i.ApplicationCommandData().Options
	if len(options) < 2 {
		discord.RespondEphemeral(s, i, "Both member and is_mercenary options are required.")
		return
	}

	// Parse options
	var targetUser *discordgo.User
	var isMercenary bool
	
	for _, opt := range options {
		switch opt.Name {
		case "member":
			targetUser = opt.UserValue(s)
		case "is_mercenary":
			isMercenary = opt.BoolValue()
		}
	}

	if targetUser == nil {
		discord.RespondEphemeral(s, i, "Member is required.")
		return
	}

	// Get member from database
	member, err := internal.GetMemberByDiscordUserIDIncludingInactive(dbx, i.GuildID, targetUser.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			discord.RespondEphemeral(s, i, fmt.Sprintf("Member %s is not in the roster. Add them first.", targetUser.Mention()))
			return
		}
		log.Printf("merc error: failed to get member: %v", err)
		discord.RespondEphemeral(s, i, "Failed to retrieve member information. Please try again.")
		return
	}

	// Update mercenary status
	err = internal.SetMemberMercenary(dbx, member.ID, isMercenary)
	if err != nil {
		log.Printf("merc error: failed to update mercenary status: %v", err)
		discord.RespondEphemeral(s, i, "Failed to update mercenary status. Please try again.")
		return
	}

	// Send success message
	statusText := "not a mercenary"
	if isMercenary {
		statusText = "a mercenary"
	}
	discord.RespondText(s, i, fmt.Sprintf("Successfully marked %s (%s) as %s.", targetUser.Mention(), member.FamilyName, statusText))
}
