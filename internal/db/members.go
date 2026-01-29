package db

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	sqlcdb "PanickedBot/internal/db/sqlc"
)

// Member represents a roster member
type Member struct {
	ID             int64
	DiscordGuildID string
	DiscordUserID  sql.NullString
	FamilyName     string
	DisplayName    sql.NullString
	Class          sql.NullString
	Spec           sql.NullString
	AP             sql.NullInt32
	AAP            sql.NullInt32
	DP             sql.NullInt32
	Evasion        sql.NullInt32
	DR             sql.NullInt32
	DRR            sql.NullFloat64
	Accuracy       sql.NullInt32
	HP             sql.NullInt32
	TotalAP        sql.NullInt32
	TotalAAP       sql.NullInt32
	MeetsCap       bool
	IsException    bool
	IsMercenary    bool
	IsActive       bool
	CreatedAt      time.Time
}

// MemberForAttendance represents a member with limited fields for attendance checking
type MemberForAttendance struct {
	ID             int64
	DiscordGuildID string
	DiscordUserID  sql.NullString
	FamilyName     string
	DisplayName    sql.NullString
	Class          sql.NullString
	Spec           sql.NullString
	AP             sql.NullInt32
	AAP            sql.NullInt32
	DP             sql.NullInt32
	CreatedAt      time.Time
}

// GetMemberByDiscordUserID retrieves a member by Discord user ID
func GetMemberByDiscordUserID(db *DB, guildID, userID string) (*Member, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	row, err := db.Queries.GetMemberByDiscordUserID(ctx, sqlcdb.GetMemberByDiscordUserIDParams{
		DiscordGuildID: guildID,
		DiscordUserID:  sql.NullString{String: userID, Valid: true},
	})
	if err != nil {
		return nil, err
	}

	return convertToMember(row), nil
}

// GetMemberByFamilyName retrieves a member by family name
func GetMemberByFamilyName(db *DB, guildID, familyName string) (*Member, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	row, err := db.Queries.GetMemberByFamilyName(ctx, sqlcdb.GetMemberByFamilyNameParams{
		DiscordGuildID: guildID,
		FamilyName:     familyName,
	})
	if err != nil {
		return nil, err
	}

	return convertToMember(row), nil
}

// GetMemberByDiscordUserIDIncludingInactive retrieves a member including inactive ones
func GetMemberByDiscordUserIDIncludingInactive(db *DB, guildID, userID string) (*Member, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	row, err := db.Queries.GetMemberByDiscordUserIDIncludingInactive(ctx, sqlcdb.GetMemberByDiscordUserIDIncludingInactiveParams{
		DiscordGuildID: guildID,
		DiscordUserID:  sql.NullString{String: userID, Valid: true},
	})
	if err != nil {
		return nil, err
	}

	return convertToMember(row), nil
}

// GetMemberByFamilyNameIncludingInactive retrieves a member including inactive ones
func GetMemberByFamilyNameIncludingInactive(db *DB, guildID, familyName string) (*Member, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	row, err := db.Queries.GetMemberByFamilyNameIncludingInactive(ctx, sqlcdb.GetMemberByFamilyNameIncludingInactiveParams{
		DiscordGuildID: guildID,
		FamilyName:     familyName,
	})
	if err != nil {
		return nil, err
	}

	return convertToMember(row), nil
}

// GetMemberByID retrieves a member by ID
func GetMemberByID(db *DB, memberID int64, guildID string) (*MemberForAttendance, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	row, err := db.Queries.GetMemberByID(ctx, sqlcdb.GetMemberByIDParams{
		ID:             uint64(memberID),
		DiscordGuildID: guildID,
	})
	if err != nil {
		return nil, err
	}

	return &MemberForAttendance{
		ID:             int64(row.ID),
		DiscordGuildID: row.DiscordGuildID,
		DiscordUserID:  row.DiscordUserID,
		FamilyName:     row.FamilyName,
		DisplayName:    row.DisplayName,
		Class:          row.Class,
		Spec:           row.Spec,
		AP:             row.Ap,
		AAP:            row.Aap,
		DP:             row.Dp,
		CreatedAt:      row.CreatedAt,
	}, nil
}

// GetAllActiveMembers retrieves all active members
func GetAllActiveMembers(db *DB, guildID string) ([]Member, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := db.Queries.GetAllActiveMembers(ctx, guildID)
	if err != nil {
		return nil, err
	}

	members := make([]Member, len(rows))
	for i, row := range rows {
		members[i] = *convertToMember(row)
	}

	return members, nil
}

// GetAllActiveMembersForAttendance retrieves all active members for attendance checking
func GetAllActiveMembersForAttendance(db *DB, guildID string) ([]MemberForAttendance, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := db.Queries.GetAllActiveMembersForAttendance(ctx, guildID)
	if err != nil {
		return nil, err
	}

	members := make([]MemberForAttendance, len(rows))
	for i, row := range rows {
		members[i] = MemberForAttendance{
			ID:             int64(row.ID),
			DiscordGuildID: row.DiscordGuildID,
			DiscordUserID:  row.DiscordUserID,
			FamilyName:     row.FamilyName,
			DisplayName:    row.DisplayName,
			Class:          row.Class,
			Spec:           row.Spec,
			AP:             row.Ap,
			AAP:            row.Aap,
			DP:             row.Dp,
			CreatedAt:      row.CreatedAt,
		}
	}

	return members, nil
}

// MemberVacation represents a vacation period
type MemberVacation struct {
	ID              int64
	DiscordGuildID  string
	RosterMemberID  int64
	StartDate       time.Time
	EndDate         time.Time
	Reason          sql.NullString
	CreatedByUserID string
	CreatedAt       time.Time
}

// GetMemberVacationsForAttendance retrieves vacations for a member
func GetMemberVacationsForAttendance(db *DB, memberID int64) ([]MemberVacation, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := db.Queries.GetMemberVacationsForAttendance(ctx, uint64(memberID))
	if err != nil {
		return nil, err
	}

	vacations := make([]MemberVacation, len(rows))
	for i, row := range rows {
		vacations[i] = MemberVacation{
			ID:              int64(row.ID),
			DiscordGuildID:  row.DiscordGuildID,
			RosterMemberID:  int64(row.RosterMemberID),
			StartDate:       row.StartDate,
			EndDate:         row.EndDate,
			Reason:          row.Reason,
			CreatedByUserID: row.CreatedByUserID,
			CreatedAt:       row.CreatedAt,
		}
	}

	return vacations, nil
}

// GetMemberWarDates retrieves war participation dates for a member
func GetMemberWarDates(db *DB, guildID string, memberID int64) ([]time.Time, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dates, err := db.Queries.GetMemberWarDates(ctx, sqlcdb.GetMemberWarDatesParams{
		DiscordGuildID: guildID,
		RosterMemberID: sql.NullInt64{Int64: memberID, Valid: true},
	})
	if err != nil {
		return nil, err
	}

	return dates, nil
}

// SetMemberActive sets the active status of a member
func SetMemberActive(db *DB, memberID int64, active bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return db.Queries.SetMemberActive(ctx, sqlcdb.SetMemberActiveParams{
		IsActive: active,
		ID:       uint64(memberID),
	})
}

// SetMemberMercenary sets the mercenary status of a member
func SetMemberMercenary(db *DB, memberID int64, mercenary bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return db.Queries.SetMemberMercenary(ctx, sqlcdb.SetMemberMercenaryParams{
		IsMercenary: mercenary,
		ID:          uint64(memberID),
	})
}

// GetMemberTeamNames retrieves team names for a member
func GetMemberTeamNames(db *DB, guildID string, memberID int64) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	names, err := db.Queries.GetMemberTeamNames(ctx, sqlcdb.GetMemberTeamNamesParams{
		RosterMemberID: uint64(memberID),
		DiscordGuildID: guildID,
	})
	if err != nil {
		return "", err
	}

	if len(names) == 0 {
		return "", nil
	}

	return strings.Join(names, ", "), nil
}

// AssignMemberToTeams assigns a member to teams (replaces existing assignments)
func AssignMemberToTeams(db *DB, memberID int64, teamIDs []int64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	qtx := db.Queries.WithTx(tx.Tx)

	// Delete existing team assignments
	err = qtx.DeleteMemberTeams(ctx, uint64(memberID))
	if err != nil {
		return err
	}

	// Deduplicate team IDs
	if len(teamIDs) > 0 {
		seen := make(map[int64]bool)
		uniqueTeamIDs := []int64{}
		for _, teamID := range teamIDs {
			if !seen[teamID] {
				seen[teamID] = true
				uniqueTeamIDs = append(uniqueTeamIDs, teamID)
			}
		}

		// Insert new team assignments
		for _, teamID := range uniqueTeamIDs {
			err = qtx.InsertMemberTeam(ctx, sqlcdb.InsertMemberTeamParams{
				RosterMemberID: uint64(memberID),
				TeamID:         uint64(teamID),
			})
			if err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

// UpdateFields represents fields that can be updated
type UpdateFields struct {
	FamilyName  *string
	DisplayName *string
	Class       *string
	Spec        *string
	TeamIDs     []int64
	MeetsCap    *bool
	AP          *int
	AAP         *int
	DP          *int
}

// UpdateMember updates member fields using raw SQL (not yet migrated to sqlc)
func UpdateMember(db *DB, memberID int64, fields UpdateFields) error {
	updates := []string{}
	args := []interface{}{}

	if fields.FamilyName != nil {
		updates = append(updates, "family_name = ?")
		args = append(args, *fields.FamilyName)
	}
	if fields.DisplayName != nil {
		updates = append(updates, "display_name = ?")
		args = append(args, *fields.DisplayName)
	}
	if fields.Class != nil {
		updates = append(updates, "class = ?")
		args = append(args, *fields.Class)
	}
	if fields.Spec != nil {
		updates = append(updates, "spec = ?")
		args = append(args, *fields.Spec)
	}
	if fields.MeetsCap != nil {
		updates = append(updates, "meets_cap = ?")
		args = append(args, *fields.MeetsCap)
	}
	if fields.AP != nil {
		updates = append(updates, "ap = ?")
		args = append(args, *fields.AP)
	}
	if fields.AAP != nil {
		updates = append(updates, "aap = ?")
		args = append(args, *fields.AAP)
	}
	if fields.DP != nil {
		updates = append(updates, "dp = ?")
		args = append(args, *fields.DP)
	}

	if len(updates) == 0 {
		return fmt.Errorf("no fields to update")
	}

	args = append(args, memberID)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := db.ExecContext(ctx, `
		UPDATE roster_members 
		SET `+strings.Join(updates, ", ")+`
		WHERE id = ?
	`, args...)

	return err
}

// CreateMember creates a new roster member using raw SQL (not yet migrated to sqlc)
func CreateMember(db *DB, guildID, discordUserID, familyName string) (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := db.ExecContext(ctx, `
		INSERT INTO roster_members (discord_guild_id, discord_user_id, family_name, is_active)
		VALUES (?, ?, ?, 1)
	`, guildID, discordUserID, familyName)

	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

// GetMemberTeamIDs retrieves team IDs for a member
func GetMemberTeamIDs(db *DB, memberID int64) ([]int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	teamIDs, err := db.Queries.GetMemberTeamIDs(ctx, uint64(memberID))
	if err != nil {
		return nil, err
	}

	result := make([]int64, len(teamIDs))
	for i, id := range teamIDs {
		result[i] = int64(id)
	}

	return result, nil
}

// Helper function to convert sqlc row to Member
func convertToMember(row interface{}) *Member {
	switch r := row.(type) {
	case sqlcdb.GetMemberByDiscordUserIDRow:
		// Parse DRR from string to float64
		var drr sql.NullFloat64
		if r.Drr.Valid {
			// Parse the string to float64
			if val, err := strconv.ParseFloat(r.Drr.String, 64); err == nil {
				drr = sql.NullFloat64{Float64: val, Valid: true}
			}
		}
		
		return &Member{
			ID:             int64(r.ID),
			DiscordGuildID: r.DiscordGuildID,
			DiscordUserID:  r.DiscordUserID,
			FamilyName:     r.FamilyName,
			DisplayName:    r.DisplayName,
			Class:          r.Class,
			Spec:           r.Spec,
			AP:             r.Ap,
			AAP:            r.Aap,
			DP:             r.Dp,
			Evasion:        r.Evasion,
			DR:             r.Dr,
			DRR:            drr,
			Accuracy:       r.Accuracy,
			HP:             r.Hp,
			TotalAP:        r.TotalAp,
			TotalAAP:       r.TotalAap,
			MeetsCap:       r.MeetsCap,
			IsException:    r.IsException,
			IsMercenary:    r.IsMercenary,
			IsActive:       r.IsActive,
			CreatedAt:      r.CreatedAt,
		}
	case sqlcdb.GetMemberByFamilyNameRow:
		var drr sql.NullFloat64
		if r.Drr.Valid {
			if val, err := strconv.ParseFloat(r.Drr.String, 64); err == nil {
				drr = sql.NullFloat64{Float64: val, Valid: true}
			}
		}
		
		return &Member{
			ID:             int64(r.ID),
			DiscordGuildID: r.DiscordGuildID,
			DiscordUserID:  r.DiscordUserID,
			FamilyName:     r.FamilyName,
			DisplayName:    r.DisplayName,
			Class:          r.Class,
			Spec:           r.Spec,
			AP:             r.Ap,
			AAP:            r.Aap,
			DP:             r.Dp,
			Evasion:        r.Evasion,
			DR:             r.Dr,
			DRR:            drr,
			Accuracy:       r.Accuracy,
			HP:             r.Hp,
			TotalAP:        r.TotalAp,
			TotalAAP:       r.TotalAap,
			MeetsCap:       r.MeetsCap,
			IsException:    r.IsException,
			IsMercenary:    r.IsMercenary,
			IsActive:       r.IsActive,
			CreatedAt:      r.CreatedAt,
		}
	case sqlcdb.GetMemberByDiscordUserIDIncludingInactiveRow:
		var drr sql.NullFloat64
		if r.Drr.Valid {
			if val, err := strconv.ParseFloat(r.Drr.String, 64); err == nil {
				drr = sql.NullFloat64{Float64: val, Valid: true}
			}
		}
		
		return &Member{
			ID:             int64(r.ID),
			DiscordGuildID: r.DiscordGuildID,
			DiscordUserID:  r.DiscordUserID,
			FamilyName:     r.FamilyName,
			DisplayName:    r.DisplayName,
			Class:          r.Class,
			Spec:           r.Spec,
			AP:             r.Ap,
			AAP:            r.Aap,
			DP:             r.Dp,
			Evasion:        r.Evasion,
			DR:             r.Dr,
			DRR:            drr,
			Accuracy:       r.Accuracy,
			HP:             r.Hp,
			TotalAP:        r.TotalAp,
			TotalAAP:       r.TotalAap,
			MeetsCap:       r.MeetsCap,
			IsException:    r.IsException,
			IsMercenary:    r.IsMercenary,
			IsActive:       r.IsActive,
			CreatedAt:      r.CreatedAt,
		}
	case sqlcdb.GetMemberByFamilyNameIncludingInactiveRow:
		var drr sql.NullFloat64
		if r.Drr.Valid {
			if val, err := strconv.ParseFloat(r.Drr.String, 64); err == nil {
				drr = sql.NullFloat64{Float64: val, Valid: true}
			}
		}
		
		return &Member{
			ID:             int64(r.ID),
			DiscordGuildID: r.DiscordGuildID,
			DiscordUserID:  r.DiscordUserID,
			FamilyName:     r.FamilyName,
			DisplayName:    r.DisplayName,
			Class:          r.Class,
			Spec:           r.Spec,
			AP:             r.Ap,
			AAP:            r.Aap,
			DP:             r.Dp,
			Evasion:        r.Evasion,
			DR:             r.Dr,
			DRR:            drr,
			Accuracy:       r.Accuracy,
			HP:             r.Hp,
			TotalAP:        r.TotalAp,
			TotalAAP:       r.TotalAap,
			MeetsCap:       r.MeetsCap,
			IsException:    r.IsException,
			IsMercenary:    r.IsMercenary,
			IsActive:       r.IsActive,
			CreatedAt:      r.CreatedAt,
		}
	case sqlcdb.GetAllActiveMembersRow:
		var drr sql.NullFloat64
		if r.Drr.Valid {
			if val, err := strconv.ParseFloat(r.Drr.String, 64); err == nil {
				drr = sql.NullFloat64{Float64: val, Valid: true}
			}
		}
		
		return &Member{
			ID:             int64(r.ID),
			DiscordGuildID: r.DiscordGuildID,
			DiscordUserID:  r.DiscordUserID,
			FamilyName:     r.FamilyName,
			DisplayName:    r.DisplayName,
			Class:          r.Class,
			Spec:           r.Spec,
			AP:             r.Ap,
			AAP:            r.Aap,
			DP:             r.Dp,
			Evasion:        r.Evasion,
			DR:             r.Dr,
			DRR:            drr,
			Accuracy:       r.Accuracy,
			HP:             r.Hp,
			TotalAP:        r.TotalAp,
			TotalAAP:       r.TotalAap,
			MeetsCap:       r.MeetsCap,
			IsException:    r.IsException,
			IsMercenary:    r.IsMercenary,
			IsActive:       r.IsActive,
			CreatedAt:      r.CreatedAt,
		}
	default:
		return nil
	}
}
