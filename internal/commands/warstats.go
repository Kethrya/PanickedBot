package commands

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/jmoiron/sqlx"

	"PanickedBot/internal/discord"
)

// WarStats represents war statistics for a member
type WarStats struct {
	FamilyName      string
	TotalWars       int
	MostRecentWar   *time.Time
	TotalKills      int
	TotalDeaths     int
}

// GetWarStats retrieves war statistics for all active members
func GetWarStats(db *sqlx.DB, guildID string) ([]WarStats, error) {
	var stats []WarStats
	
	// Query to get war stats for each active member
	// Returns all active members including those with no war participation
	query := `
		SELECT 
			rm.family_name,
			COUNT(DISTINCT wl.war_id) as total_wars,
			MAX(w.war_date) as most_recent_war,
			COALESCE(SUM(wl.kills), 0) as total_kills,
			COALESCE(SUM(wl.deaths), 0) as total_deaths
		FROM roster_members rm
		LEFT JOIN war_lines wl ON rm.id = wl.roster_member_id
		LEFT JOIN wars w ON wl.war_id = w.id AND w.is_excluded = 0
		WHERE rm.discord_guild_id = ? 
		  AND rm.is_active = 1
		  AND rm.family_name IS NOT NULL
		GROUP BY rm.id, rm.family_name
		ORDER BY rm.family_name
	`
	
	rows, err := db.Query(query, guildID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	for rows.Next() {
		var stat WarStats
		var familyName sql.NullString
		var mostRecentWar sql.NullTime
		
		err := rows.Scan(&familyName, &stat.TotalWars, &mostRecentWar, &stat.TotalKills, &stat.TotalDeaths)
		if err != nil {
			return nil, err
		}
		
		if familyName.Valid {
			stat.FamilyName = familyName.String
		}
		
		if mostRecentWar.Valid {
			stat.MostRecentWar = &mostRecentWar.Time
		}
		
		stats = append(stats, stat)
	}
	
	return stats, rows.Err()
}

// formatWarStatLine formats a single war stat entry as a string
func formatWarStatLine(stat WarStats) string {
	familyName := truncateString(stat.FamilyName, 20)
	
	totalWarsStr := "N/A"
	mostRecentStr := "N/A"
	killsStr := "N/A"
	deathsStr := "N/A"
	kdStr := "N/A"
	
	if stat.TotalWars > 0 {
		totalWarsStr = fmt.Sprintf("%d", stat.TotalWars)
		killsStr = fmt.Sprintf("%d", stat.TotalKills)
		deathsStr = fmt.Sprintf("%d", stat.TotalDeaths)
		
		if stat.MostRecentWar != nil {
			mostRecentStr = stat.MostRecentWar.Format("2006-01-02")
		}
		
		// Calculate K/D ratio
		if stat.TotalDeaths > 0 {
			kd := float64(stat.TotalKills) / float64(stat.TotalDeaths)
			kdStr = fmt.Sprintf("%.2f", kd)
		} else if stat.TotalKills > 0 {
			kdStr = fmt.Sprintf("%.2f", float64(stat.TotalKills))
		} else {
			kdStr = "0.00"
		}
	}

	return fmt.Sprintf("%-20s %12s %-15s %8s %8s %8s\n",
		familyName, totalWarsStr, mostRecentStr, killsStr, deathsStr, kdStr)
}

func handleWarStats(s *discordgo.Session, i *discordgo.InteractionCreate, dbx *sqlx.DB, cfg *GuildConfig) {
	if !hasOfficerPermission(s, i, cfg) {
		discord.RespondEphemeral(s, i, "You need officer role or admin permission to use this command.")
		return
	}

	// Get war statistics
	stats, err := GetWarStats(dbx, i.GuildID)
	if err != nil {
		log.Printf("warstats error: %v", err)
		discord.RespondEphemeral(s, i, "Failed to retrieve war statistics. Please try again.")
		return
	}

	if len(stats) == 0 {
		discord.RespondEphemeral(s, i, "No active roster members found.")
		return
	}

	// Build response message with aligned columns
	var response strings.Builder
	response.WriteString("**War Statistics**\n```\n")

	// Header
	response.WriteString(fmt.Sprintf("%-20s %12s %-15s %8s %8s %8s\n", 
		"Family Name", "Total Wars", "Most Recent", "Kills", "Deaths", "K/D"))
	response.WriteString(strings.Repeat("-", 85) + "\n")

	// Data rows
	for _, stat := range stats {
		response.WriteString(formatWarStatLine(stat))
	}

	response.WriteString("```")

	// Discord has a 2000 character limit for messages
	responseText := response.String()
	if len(responseText) > 2000 {
		// If too long, show fewer rows
		var truncatedResponse strings.Builder
		truncatedResponse.WriteString("**War Statistics** (showing first entries)\n```\n")
		truncatedResponse.WriteString(fmt.Sprintf("%-20s %12s %-15s %8s %8s %8s\n",
			"Family Name", "Total Wars", "Most Recent", "Kills", "Deaths", "K/D"))
		truncatedResponse.WriteString(strings.Repeat("-", 85) + "\n")

		for _, stat := range stats {
			line := formatWarStatLine(stat)
			
			if len(truncatedResponse.String()+line+"```") > 1990 {
				break
			}
			truncatedResponse.WriteString(line)
		}
		truncatedResponse.WriteString("```")
		discord.RespondText(s, i, truncatedResponse.String())
	} else {
		discord.RespondText(s, i, responseText)
	}
}
