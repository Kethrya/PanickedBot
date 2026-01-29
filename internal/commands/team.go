package commands

import (
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/jmoiron/sqlx"

	"PanickedBot/internal/db"
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

	// Create team (will reactivate if exists and inactive)
	_, reactivated, err := db.CreateTeam(dbx, i.GuildID, teamName)
	if err == db.ErrTeamAlreadyExists {
		// Team already exists and is active
		discord.RespondEphemeral(s, i, "A team with that name already exists and is active.")
		return
	} else if err != nil {
		log.Printf("add team error: %v", err)
		discord.RespondEphemeral(s, i, "Failed to create team. Please try again.")
		return
	}

	if reactivated {
		discord.RespondText(s, i, "Team **"+teamName+"** reactivated successfully.")
	} else {
		discord.RespondText(s, i, "Team **"+teamName+"** created successfully.")
	}
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

	found, err := db.DeactivateTeam(dbx, i.GuildID, teamName)
	if err != nil {
		log.Printf("delete team error: %v", err)
		discord.RespondEphemeral(s, i, "Failed to delete team. Please try again.")
		return
	}

	if !found {
		discord.RespondEphemeral(s, i, "Team not found or already deleted.")
		return
	}

	discord.RespondText(s, i, "Team **"+teamName+"** deleted successfully.")
}
