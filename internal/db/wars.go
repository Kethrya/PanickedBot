package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

// WarStats represents war statistics for a member
type WarStats struct {
	FamilyName    string
	TotalWars     int
	MostRecentWar *time.Time
	TotalKills    int
	TotalDeaths   int
}

// GetWarStats retrieves war statistics for all active members
func GetWarStats(db *sqlx.DB, guildID string) ([]WarStats, error) {
	var stats []WarStats

	// Query to get war stats for each active member
	// Only count wars, kills, and deaths from non-excluded wars
	query := `
		SELECT 
			rm.family_name,
			COUNT(DISTINCT CASE WHEN w.id IS NOT NULL THEN w.id END) as total_wars,
			MAX(CASE WHEN w.id IS NOT NULL THEN w.war_date END) as most_recent_war,
			COALESCE(SUM(CASE WHEN w.id IS NOT NULL THEN wl.kills ELSE 0 END), 0) as total_kills,
			COALESCE(SUM(CASE WHEN w.id IS NOT NULL THEN wl.deaths ELSE 0 END), 0) as total_deaths
		FROM roster_members rm
		LEFT JOIN war_lines wl ON rm.id = wl.roster_member_id
		LEFT JOIN wars w ON wl.war_id = w.id AND w.is_excluded = 0
		WHERE rm.discord_guild_id = ? 
		  AND rm.is_active = 1
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
		var familyName string
		var mostRecentWar sql.NullTime

		err := rows.Scan(&familyName, &stat.TotalWars, &mostRecentWar, &stat.TotalKills, &stat.TotalDeaths)
		if err != nil {
			return nil, err
		}

		stat.FamilyName = familyName

		if mostRecentWar.Valid {
			stat.MostRecentWar = &mostRecentWar.Time
		}

		stats = append(stats, stat)
	}

	return stats, rows.Err()
}

// WarLineData represents a single war line entry
type WarLineData struct {
	FamilyName string
	Kills      int
	Deaths     int
}

// CreateWarFromCSV creates a war entry and associated war lines from CSV data
func CreateWarFromCSV(db *sqlx.DB, guildID string, requestChannelID string, requestMessageID string, requestedByUserID string, warDate time.Time, warLines []WarLineData) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Create war_job entry
	jobResult, err := tx.ExecContext(ctx, `
		INSERT INTO war_jobs (discord_guild_id, request_channel_id, request_message_id, 
		                      requested_by_user_id, status, started_at, finished_at)
		VALUES (?, ?, ?, ?, 'done', NOW(), NOW())
	`, guildID, requestChannelID, requestMessageID, requestedByUserID)
	if err != nil {
		return fmt.Errorf("failed to create war job: %w", err)
	}

	jobID, err := jobResult.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get job ID: %w", err)
	}

	// Create war entry
	warResult, err := tx.ExecContext(ctx, `
		INSERT INTO wars (discord_guild_id, job_id, war_date, label)
		VALUES (?, ?, ?, ?)
	`, guildID, jobID, warDate.Format("2006-01-02"), fmt.Sprintf("CSV Import - %s", warDate.Format("2006-01-02")))
	if err != nil {
		return fmt.Errorf("failed to create war: %w", err)
	}

	warID, err := warResult.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get war ID: %w", err)
	}

	// Create war_lines entries
	for _, line := range warLines {
		// Try to match the family name to a roster member (case insensitive)
		var rosterMemberID sql.NullInt64
		err := tx.GetContext(ctx, &rosterMemberID, `
			SELECT id FROM roster_members
			WHERE discord_guild_id = ? AND LOWER(family_name) = LOWER(?)
			LIMIT 1
		`, guildID, line.FamilyName)

		if err == sql.ErrNoRows {
			// Roster member doesn't exist - create one
			result, err := tx.ExecContext(ctx, `
				INSERT INTO roster_members (discord_guild_id, family_name, is_active)
				VALUES (?, ?, 1)
			`, guildID, line.FamilyName)
			if err != nil {
				return fmt.Errorf("failed to create roster member for '%s': %w", line.FamilyName, err)
			}

			newID, err := result.LastInsertId()
			if err != nil {
				return fmt.Errorf("failed to get new roster member ID for '%s': %w", line.FamilyName, err)
			}

			rosterMemberID.Int64 = newID
			rosterMemberID.Valid = true
		} else if err != nil {
			return fmt.Errorf("failed to lookup roster member for '%s': %w", line.FamilyName, err)
		}

		// Insert war_line
		_, err = tx.ExecContext(ctx, `
			INSERT INTO war_lines (war_id, roster_member_id, ocr_name, kills, deaths, matched_name)
			VALUES (?, ?, ?, ?, ?, ?)
		`, warID, rosterMemberID, line.FamilyName, line.Kills, line.Deaths, line.FamilyName)
		if err != nil {
			return fmt.Errorf("failed to create war line for '%s': %w", line.FamilyName, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
