package commands

import (
	"context"
	"database/sql"
	"log"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/jmoiron/sqlx"

	"PanickedBot/internal/discord"
)

func handleAddTeam(s *discordgo.Session, i *discordgo.InteractionCreate, dbx *sqlx.DB, cfg *GuildConfig) {
	if !hasOfficerPermission(s, i, cfg) {
		discord.RespondEphemeral(s, i, "You need officer role or admin permission to use this command.")
		return
	}

	// Parse options
	var teamName string
	for _, opt := range i.ApplicationCommandData().Options {
		if opt.Name == "name" {
			teamName = opt.StringValue()
		}
	}

	if teamName == "" {
		discord.RespondEphemeral(s, i, "Team name is required.")
		return
	}

	// Generate code from name (lowercase, replace spaces with underscores)
	code := strings.ToLower(strings.ReplaceAll(teamName, " ", "_"))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Check if team already exists
	var existingTeam struct {
		ID       int64 `db:"id"`
		IsActive bool  `db:"is_active"`
	}
	err := dbx.GetContext(ctx, &existingTeam, `
		SELECT id, is_active FROM `+"teams"+`
		WHERE discord_guild_id = ? AND (code = ? OR display_name = ?)
	`, i.GuildID, code, teamName)

	if err == nil {
		// Team exists - check if it's inactive
		if !existingTeam.IsActive {
			// Reactivate the team
			_, err := dbx.ExecContext(ctx, `
				UPDATE `+"teams"+`
				SET is_active = 1
				WHERE id = ?
			`, existingTeam.ID)
			
			if err != nil {
				log.Printf("reactivate team error: %v", err)
				discord.RespondEphemeral(s, i, "Failed to reactivate team. Please try again.")
				return
			}
			
			discord.RespondText(s, i, "Team **"+teamName+"** reactivated successfully.")
			return
		} else {
			// Team is already active
			discord.RespondEphemeral(s, i, "A team with that name already exists and is active.")
			return
		}
	} else if err != sql.ErrNoRows {
		log.Printf("check existing team error: %v", err)
		discord.RespondEphemeral(s, i, "Failed to create team. Please try again.")
		return
	}

	// Team doesn't exist - create it
	_, err = dbx.ExecContext(ctx, `
		INSERT INTO `+"teams"+` (discord_guild_id, code, display_name, is_active)
		VALUES (?, ?, ?, 1)
	`, i.GuildID, code, teamName)

	if err != nil {
		log.Printf("add team error: %v", err)
		discord.RespondEphemeral(s, i, "Failed to create team. Please try again.")
		return
	}

	discord.RespondText(s, i, "Team **"+teamName+"** created successfully.")
}

func handleDeleteTeam(s *discordgo.Session, i *discordgo.InteractionCreate, dbx *sqlx.DB, cfg *GuildConfig) {
	if !hasOfficerPermission(s, i, cfg) {
		discord.RespondEphemeral(s, i, "You need officer role or admin permission to use this command.")
		return
	}

	// Parse options
	var teamName string
	for _, opt := range i.ApplicationCommandData().Options {
		if opt.Name == "name" {
			teamName = opt.StringValue()
		}
	}

	if teamName == "" {
		discord.RespondEphemeral(s, i, "Team name is required.")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := dbx.ExecContext(ctx, `
		UPDATE `+"teams"+`
		SET is_active = 0
		WHERE discord_guild_id = ? AND display_name = ? AND is_active = 1
	`, i.GuildID, teamName)

	if err != nil {
		log.Printf("delete team error: %v", err)
		discord.RespondEphemeral(s, i, "Failed to delete team. Please try again.")
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		discord.RespondEphemeral(s, i, "Team not found or already deleted.")
		return
	}

	discord.RespondText(s, i, "Team **"+teamName+"** deleted successfully.")
}
