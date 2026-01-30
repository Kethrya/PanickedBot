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

func handleVacation(s *discordgo.Session, i *discordgo.InteractionCreate, dbx *db.DB, cfg *GuildConfig) {
	if !hasOfficerPermission(s, i, cfg) {
		discord.RespondEphemeral(s, i, "You need officer role or admin permission to use this command.")
		return
	}

	options := i.ApplicationCommandData().Options
	if len(options) < 3 {
		discord.RespondEphemeral(s, i, "Member, start_date, and end_date are required.")
		return
	}

	// Parse options
	var targetUser *discordgo.User
	var startDateStr, endDateStr, reason string
	
	for _, opt := range options {
		switch opt.Name {
		case "member":
			targetUser = opt.UserValue(s)
		case "start_date":
			startDateStr = opt.StringValue()
		case "end_date":
			endDateStr = opt.StringValue()
		case "reason":
			reason = opt.StringValue()
		}
	}

	if targetUser == nil {
		discord.RespondEphemeral(s, i, "Member is required.")
		return
	}

	// Get Eastern timezone
	est := getEasternLocation()

	// Parse dates in Eastern timezone
	startDate, err := parseFlexibleDate(startDateStr, est)
	if err != nil {
		discord.RespondEphemeral(s, i, "Invalid start date format. Use DD-MM-YY (e.g., 25-12-24).")
		return
	}

	endDate, err := parseFlexibleDate(endDateStr, est)
	if err != nil {
		discord.RespondEphemeral(s, i, "Invalid end date format. Use DD-MM-YY (e.g., 31-12-24).")
		return
	}

	// Validate date range
	if endDate.Before(startDate) {
		discord.RespondEphemeral(s, i, "End date must be on or after start date.")
		return
	}

	// Get member from database
	member, err := internal.GetMemberByDiscordUserIDIncludingInactive(dbx, i.GuildID, targetUser.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			discord.RespondEphemeral(s, i, fmt.Sprintf("Member %s is not in the roster. Add them first.", targetUser.Mention()))
			return
		}
		log.Printf("vacation error: failed to get member: %v", err)
		discord.RespondEphemeral(s, i, "Failed to retrieve member information. Please try again.")
		return
	}

	// Create vacation entry
	_, err = db.CreateVacation(dbx, i.GuildID, member.ID, startDate, endDate, reason, i.Member.User.ID)
	if err != nil {
		log.Printf("vacation error: failed to create vacation: %v", err)
		discord.RespondEphemeral(s, i, "Failed to create vacation entry. Please try again.")
		return
	}

	// Send success message
	reasonText := ""
	if reason != "" {
		reasonText = fmt.Sprintf(" (Reason: %s)", reason)
	}
	discord.RespondText(s, i, fmt.Sprintf("Successfully added vacation for %s (%s) from %s to %s%s.",
		targetUser.Mention(),
		member.FamilyName,
		startDate.Format("02-01-06"),
		endDate.Format("02-01-06"),
		reasonText))
}
