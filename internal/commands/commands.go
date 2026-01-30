package commands

import (
	"database/sql"
	"errors"
	"log"

	"github.com/bwmarrin/discordgo"

	"PanickedBot/internal"
	"PanickedBot/internal/db"
	"PanickedBot/internal/discord"
)

// float64Ptr returns a pointer to a float64 value
func float64Ptr(f float64) *float64 {
	return &f
}

// Spec choices
func getSpecChoices() []*discordgo.ApplicationCommandOptionChoice {
	return []*discordgo.ApplicationCommandOptionChoice{
		{Name: "Succession", Value: "succession"},
		{Name: "Awakening", Value: "awakening"},
		{Name: "Ascension", Value: "ascension"},
	}
}

// GetCommands returns all application commands
func GetCommands() []*discordgo.ApplicationCommand {
	return []*discordgo.ApplicationCommand{
		{Name: "ping", Description: "health check"},
		setupCommand(),
		{
			Name:        "addteam",
			Description: "Add a new team (officer role required)",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "name",
					Description: "Team name",
					Required:    true,
				},
			},
		},
		{
			Name:        "deleteteam",
			Description: "Delete an existing team (officer role required)",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "name",
					Description: "Team name to delete",
					Required:    true,
				},
			},
		},
		{
			Name:        "inactive",
			Description: "Mark a member as inactive (officer role required)",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionUser,
					Name:        "member",
					Description: "Discord member to mark as inactive",
					Required:    false,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "family_name",
					Description: "Family name of member to mark as inactive",
					Required:    false,
				},
			},
		},
		{
			Name:        "updateself",
			Description: "Update your own member information (guild member role required)",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "family_name",
					Description: "Your family name in BDO",
					Required:    false,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "class",
					Description: "Your BDO class",
					Required:    false,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "spec",
					Description: "Your class specialization",
					Required:    false,
					Choices:     getSpecChoices(),
				},
			},
		},
		{
			Name:        "gear",
			Description: "Update gear stats (your own or another member's if you're an officer)",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionInteger,
					Name:        "ap",
					Description: "Attack Power (AP)",
					Required:    true,
					MinValue:    float64Ptr(0),
				},
				{
					Type:        discordgo.ApplicationCommandOptionInteger,
					Name:        "aap",
					Description: "Awakening Attack Power (AAP)",
					Required:    true,
					MinValue:    float64Ptr(0),
				},
				{
					Type:        discordgo.ApplicationCommandOptionInteger,
					Name:        "dp",
					Description: "Defense Power (DP)",
					Required:    true,
					MinValue:    float64Ptr(0),
				},
				{
					Type:        discordgo.ApplicationCommandOptionUser,
					Name:        "member",
					Description: "Discord member to update (officers only, leave empty to update yourself)",
					Required:    false,
				},
			},
		},
		{
			Name:        "updatemember",
			Description: "Update another member's information (officer role required)",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionUser,
					Name:        "member",
					Description: "Discord member to update",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "family_name",
					Description: "Member's family name in BDO",
					Required:    false,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "class",
					Description: "Member's BDO class",
					Required:    false,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "spec",
					Description: "Member's class specialization",
					Required:    false,
					Choices:     getSpecChoices(),
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "teams",
					Description: "Comma-separated team names to assign the member to (replaces existing teams)",
					Required:    false,
				},
				{
					Type:        discordgo.ApplicationCommandOptionBoolean,
					Name:        "meets_cap",
					Description: "Whether member meets required stat caps",
					Required:    false,
				},
			},
		},
		{
			Name:        "active",
			Description: "Mark a member as active (officer role required)",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionUser,
					Name:        "member",
					Description: "Discord member to mark as active",
					Required:    false,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "family_name",
					Description: "Family name of member to mark as active",
					Required:    false,
				},
			},
		},
		{
			Name:        "roster",
			Description: "Get all roster member information (officer role required)",
		},
		{
			Name:        "merc",
			Description: "Mark a member as mercenary or not (officer role required)",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionUser,
					Name:        "member",
					Description: "Discord member to update",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionBoolean,
					Name:        "is_mercenary",
					Description: "Whether the member is a mercenary",
					Required:    true,
				},
			},
		},
		{
			Name:        "vacation",
			Description: "Add a vacation period for a member (officer role required)",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionUser,
					Name:        "member",
					Description: "Discord member going on vacation",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "start_date",
					Description: "Vacation start date (DD-MM-YY)",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "end_date",
					Description: "Vacation end date (DD-MM-YY)",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "reason",
					Description: "Optional reason for vacation",
					Required:    false,
				},
			},
		},
		{
			Name:        "warstats",
			Description: "Get war statistics for all roster members (officer role required)",
		},
		{
			Name:        "warresults",
			Description: "Get results of all wars from most recent to oldest (officer role required)",
		},
		{
			Name:        "removewar",
			Description: "Remove war data for a specific date (officer role required)",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "date",
					Description: "War date in DD-MM-YY format",
					Required:    true,
				},
			},
		},
		{
			Name:        "addwar",
			Description: "Import war data from a CSV or image file (officer role required)",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionAttachment,
					Name:        "file",
					Description: "CSV or image file (<5MB) with war data",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "result",
					Description: "War result",
					Required:    true,
					Choices: []*discordgo.ApplicationCommandOptionChoice{
						{Name: "Win", Value: "win"},
						{Name: "Lose", Value: "lose"},
					},
				},
			},
		},
		{
			Name:        "attendance",
			Description: "Get all members with attendance problems (officer role required)",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionInteger,
					Name:        "weeks",
					Description: "Number of weeks to check (default: 4)",
					Required:    false,
					MinValue:    float64Ptr(1),
					MaxValue:    52,
				},
			},
		},
		{
			Name:        "checkattendance",
			Description: "Check attendance for a specific member (officer role required)",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionUser,
					Name:        "member",
					Description: "Discord member to check",
					Required:    false,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "family_name",
					Description: "Family name of member to check",
					Required:    false,
				},
				{
					Type:        discordgo.ApplicationCommandOptionInteger,
					Name:        "weeks",
					Description: "Number of weeks to check (default: 4)",
					Required:    false,
					MinValue:    float64Ptr(1),
					MaxValue:    52,
				},
			},
		},
	}
}

// CreateInteractionHandler creates the interaction handler for commands
func CreateInteractionHandler(database *db.DB) func(s *discordgo.Session, i *discordgo.InteractionCreate) {
	return func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if i.Type != discordgo.InteractionApplicationCommand {
			return
		}

		cmdName := i.ApplicationCommandData().Name

		if cmdName == "setup" {
			handleSetup(s, i, database)
			return
		}

		if i.GuildID == "" {
			discord.RespondText(s, i, "This bot only works in servers.")
			return
		}

		cfg, err := internal.LoadGuildConfig(database, i.GuildID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				discord.RespondEphemeral(s, i, "Guild is not set up yet. Run /setup first.")
				return
			}
			log.Printf("load guild config: %v", err)
			discord.RespondEphemeral(s, i, "Failed to load guild configuration. Please try again.")
			return
		}

		// Channel guard (command channel)
		if cfg.CommandChannelID != "" && i.ChannelID != cfg.CommandChannelID {
			discord.RespondEphemeral(
				s,
				i,
				"Use this command in <#"+cfg.CommandChannelID+">.",
			)
			return
		}

		switch i.ApplicationCommandData().Name {

		case "ping":
			discord.RespondText(s, i, "pong")

		case "addteam":
			handleAddTeam(s, i, database, cfg)

		case "deleteteam":
			handleDeleteTeam(s, i, database, cfg)

		case "updateself":
			handleUpdateSelf(s, i, database, cfg)

		case "gear":
			handleGear(s, i, database, cfg)

		case "updatemember":
			handleUpdateMember(s, i, database, cfg)

		case "inactive":
			handleInactive(s, i, database, cfg)

		case "active":
			handleActive(s, i, database, cfg)

		case "roster":
			handleGetRoster(s, i, database, cfg)

		case "merc":
			handleMerc(s, i, database, cfg)

		case "vacation":
			handleVacation(s, i, database, cfg)

		case "warstats":
			handleWarStats(s, i, database, cfg)

		case "warresults":
			handleWarResults(s, i, database, cfg)

		case "removewar":
			handleRemoveWar(s, i, database, cfg)

		case "addwar":
			handleAddWar(s, i, database, cfg)

		case "attendance":
			handleAttendance(s, i, database, cfg)

		case "checkattendance":
			handleCheckAttendance(s, i, database, cfg)

		default:
			discord.RespondEphemeral(s, i, "Unknown command.")
		}

	}
}
