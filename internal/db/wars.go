package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	sqlcdb "PanickedBot/internal/db/sqlc"
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
func GetWarStats(db *DB, guildID string) ([]WarStats, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := db.Queries.GetWarStats(ctx, guildID)
	if err != nil {
		return nil, err
	}

	stats := make([]WarStats, 0, len(rows))
	for _, row := range rows {
		stat := WarStats{
			FamilyName:  row.FamilyName,
			TotalWars:   int(row.TotalWars),
			TotalKills:  int(row.TotalKills),
			TotalDeaths: int(row.TotalDeaths),
		}

		// Handle most_recent_war which can be NULL (returned as interface{})
		if row.MostRecentWar != nil {
			if t, ok := row.MostRecentWar.(time.Time); ok {
				stat.MostRecentWar = &t
			}
		}

		stats = append(stats, stat)
	}

	return stats, nil
}

// WarLineData represents a single war line entry
type WarLineData struct {
	FamilyName string
	Kills      int
	Deaths     int
}

// CreateWarFromCSV creates a war entry and associated war lines from CSV data
func CreateWarFromCSV(db *DB, guildID string, requestChannelID string, requestMessageID string, requestedByUserID string, warDate time.Time, warLines []WarLineData) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Create queries with transaction
	qtx := db.Queries.WithTx(tx.Tx)

	// Create war_job entry
	jobResult, err := qtx.CreateWarJob(ctx, sqlcdb.CreateWarJobParams{
		DiscordGuildID:    guildID,
		RequestChannelID:  requestChannelID,
		RequestMessageID:  requestMessageID,
		RequestedByUserID: requestedByUserID,
	})
	if err != nil {
		return fmt.Errorf("failed to create war job: %w", err)
	}

	jobID, err := jobResult.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get job ID: %w", err)
	}

	// Create war entry
	warResult, err := qtx.CreateWar(ctx, sqlcdb.CreateWarParams{
		DiscordGuildID: guildID,
		JobID:          uint64(jobID),
		WarDate:        warDate,
		Label:          sql.NullString{String: fmt.Sprintf("CSV Import - %s", warDate.Format("2006-01-02")), Valid: true},
	})
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
		rosterMemberID, err := qtx.GetRosterMemberByFamilyName(ctx, sqlcdb.GetRosterMemberByFamilyNameParams{
			DiscordGuildID: guildID,
			LOWER:          line.FamilyName,
		})

		var memberID sql.NullInt64
		if err == sql.ErrNoRows {
			// Roster member doesn't exist - create one
			result, err := qtx.CreateRosterMember(ctx, sqlcdb.CreateRosterMemberParams{
				DiscordGuildID: guildID,
				FamilyName:     line.FamilyName,
			})
			if err != nil {
				return fmt.Errorf("failed to create roster member for '%s': %w", line.FamilyName, err)
			}

			newID, err := result.LastInsertId()
			if err != nil {
				return fmt.Errorf("failed to get new roster member ID for '%s': %w", line.FamilyName, err)
			}

			memberID.Int64 = newID
			memberID.Valid = true
		} else if err != nil {
			return fmt.Errorf("failed to lookup roster member for '%s': %w", line.FamilyName, err)
		} else {
			memberID.Int64 = int64(rosterMemberID)
			memberID.Valid = true
		}

		// Insert war_line
		err = qtx.CreateWarLine(ctx, sqlcdb.CreateWarLineParams{
			WarID:          uint64(warID),
			RosterMemberID: memberID,
			OcrName:        line.FamilyName,
			Kills:          int32(line.Kills),
			Deaths:         int32(line.Deaths),
			MatchedName:    sql.NullString{String: line.FamilyName, Valid: true},
		})
		if err != nil {
			return fmt.Errorf("failed to create war line for '%s': %w", line.FamilyName, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
