package member

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
)

// Member represents a roster member
type Member struct {
	ID             int64  `db:"id"`
	DiscordGuildID string `db:"discord_guild_id"`
	DiscordUserID  string `db:"discord_user_id"`
	BDOName        string `db:"bdo_name"`
	FamilyName     string `db:"family_name"`
	Class          string `db:"class"`
	Spec           string `db:"spec"`
	GroupID        *int64 `db:"group_id"`
	
	// Combat stats
	AP       *int `db:"ap"`
	AAP      *int `db:"aap"`
	DP       *int `db:"dp"`
	Evasion  *int `db:"evasion"`
	DR       *int `db:"dr"`
	DRR      *float64 `db:"drr"`
	Accuracy *int `db:"accuracy"`
	HP       *int `db:"hp"`
	TotalAP  *int `db:"total_ap"`
	TotalAAP *int `db:"total_aap"`
	
	// Status flags
	MeetsCap    bool `db:"meets_cap"`
	IsException bool `db:"is_exception"`
	IsActive    bool `db:"is_active"`
}

// UpdateFields represents fields that can be updated
type UpdateFields struct {
	FamilyName *string
	Class      *string
	Spec       *string
	GroupID    *int64
	MeetsCap   *bool
}

// GetByDiscordUserID retrieves a member by Discord user ID
func GetByDiscordUserID(db *sqlx.DB, guildID, userID string) (*Member, error) {
	var m Member
	err := db.Get(&m, `
		SELECT id, discord_guild_id, discord_user_id, bdo_name, family_name, 
		       class, spec, group_id, ap, aap, dp, evasion, dr, drr, 
		       accuracy, hp, total_ap, total_aap, meets_cap, is_exception, is_active
		FROM roster_members 
		WHERE discord_guild_id = ? AND discord_user_id = ? AND is_active = 1
	`, guildID, userID)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// GetByFamilyName retrieves a member by BDO family name
func GetByFamilyName(db *sqlx.DB, guildID, familyName string) (*Member, error) {
	var m Member
	err := db.Get(&m, `
		SELECT id, discord_guild_id, discord_user_id, bdo_name, family_name, 
		       class, spec, group_id, ap, aap, dp, evasion, dr, drr, 
		       accuracy, hp, total_ap, total_aap, meets_cap, is_exception, is_active
		FROM roster_members 
		WHERE discord_guild_id = ? AND family_name = ? AND is_active = 1
	`, guildID, familyName)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// GetByDiscordUserIDOrFamilyName retrieves a member by Discord user ID or family name
func GetByDiscordUserIDOrFamilyName(db *sqlx.DB, guildID, discordUserID, familyName string) (*Member, error) {
	if discordUserID != "" {
		return GetByDiscordUserID(db, guildID, discordUserID)
	}
	if familyName != "" {
		return GetByFamilyName(db, guildID, familyName)
	}
	return nil, sql.ErrNoRows
}

// GetByDiscordUserIDIncludingInactive retrieves a member by Discord user ID, including inactive members
func GetByDiscordUserIDIncludingInactive(db *sqlx.DB, guildID, userID string) (*Member, error) {
	var m Member
	err := db.Get(&m, `
		SELECT id, discord_guild_id, discord_user_id, bdo_name, family_name, 
		       class, spec, group_id, ap, aap, dp, evasion, dr, drr, 
		       accuracy, hp, total_ap, total_aap, meets_cap, is_exception, is_active
		FROM roster_members 
		WHERE discord_guild_id = ? AND discord_user_id = ?
	`, guildID, userID)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// GetByFamilyNameIncludingInactive retrieves a member by BDO family name, including inactive members
func GetByFamilyNameIncludingInactive(db *sqlx.DB, guildID, familyName string) (*Member, error) {
	var m Member
	err := db.Get(&m, `
		SELECT id, discord_guild_id, discord_user_id, bdo_name, family_name, 
		       class, spec, group_id, ap, aap, dp, evasion, dr, drr, 
		       accuracy, hp, total_ap, total_aap, meets_cap, is_exception, is_active
		FROM roster_members 
		WHERE discord_guild_id = ? AND family_name = ?
	`, guildID, familyName)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// Update updates member fields
func Update(db *sqlx.DB, memberID int64, fields UpdateFields) error {
	updates := []string{}
	args := []interface{}{}
	
	if fields.FamilyName != nil {
		updates = append(updates, "family_name = ?")
		args = append(args, *fields.FamilyName)
	}
	if fields.Class != nil {
		updates = append(updates, "class = ?")
		args = append(args, *fields.Class)
	}
	if fields.Spec != nil {
		updates = append(updates, "spec = ?")
		args = append(args, *fields.Spec)
	}
	if fields.GroupID != nil {
		updates = append(updates, "group_id = ?")
		args = append(args, *fields.GroupID)
	}
	if fields.MeetsCap != nil {
		updates = append(updates, "meets_cap = ?")
		args = append(args, *fields.MeetsCap)
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

// SetActive sets the is_active flag for a member
func SetActive(db *sqlx.DB, memberID int64, active bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	_, err := db.ExecContext(ctx, `
		UPDATE roster_members 
		SET is_active = ?
		WHERE id = ?
	`, active, memberID)
	
	return err
}
