package db

import (
	"context"
	"database/sql"
	"time"

	sqlcdb "PanickedBot/internal/db/sqlc"
)

// Vacation represents a vacation entry
type Vacation struct {
	ID              int64
	RosterMemberID  int64
	StartDate       time.Time
	EndDate         time.Time
	Reason          string
	CreatedByUserID string
	CreatedAt       time.Time
}

// CreateVacation creates a new vacation entry for a member
func CreateVacation(db *DB, guildID string, memberID int64, startDate, endDate time.Time, reason string, createdByUserID string) (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	reasonNullString := sql.NullString{Valid: false}
	if reason != "" {
		reasonNullString = sql.NullString{String: reason, Valid: true}
	}

	result, err := db.Queries.CreateVacation(ctx, sqlcdb.CreateVacationParams{
		DiscordGuildID:   guildID,
		RosterMemberID:   uint64(memberID),
		StartDate:        startDate,
		EndDate:          endDate,
		Reason:           reasonNullString,
		CreatedByUserID:  createdByUserID,
	})
	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

// GetMemberVacations retrieves all vacations for a member
func GetMemberVacations(db *DB, memberID int64) ([]Vacation, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := db.Queries.GetMemberVacations(ctx, uint64(memberID))
	if err != nil {
		return nil, err
	}

	vacations := make([]Vacation, 0, len(rows))
	for _, row := range rows {
		vacation := Vacation{
			ID:              int64(row.ID),
			RosterMemberID:  int64(row.RosterMemberID),
			StartDate:       row.StartDate,
			EndDate:         row.EndDate,
			Reason:          row.Reason.String,
			CreatedByUserID: row.CreatedByUserID,
			CreatedAt:       row.CreatedAt,
		}
		vacations = append(vacations, vacation)
	}

	return vacations, nil
}
