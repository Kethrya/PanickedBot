package db

import (
	"context"
	"database/sql"
	"time"

	sqlcdb "PanickedBot/internal/db/sqlc"
)

// GuildConfig represents guild configuration
type GuildConfig struct {
	CommandChannelID  string  `db:"command_channel_id"`
	OfficerRoleID     *string `db:"officer_role_id"`
	GuildMemberRoleID *string `db:"guild_member_role_id"`
	MercenaryRoleID   *string `db:"mercenary_role_id"`
}

// UpsertGuildAndConfig creates or updates guild and configuration in a transaction
func UpsertGuildAndConfig(db *DB, guildID, guildName, commandChannelID string, officerRoleID, guildMemberRoleID, mercenaryRoleID *string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	// Create queries with transaction
	qtx := db.Queries.WithTx(tx.Tx)

	// Prepare guild name value
	var guildNameValue sql.NullString
	if guildName != "" {
		guildNameValue = sql.NullString{String: guildName, Valid: true}
	}

	// Upsert guild row
	err = qtx.UpsertGuild(ctx, sqlcdb.UpsertGuildParams{
		DiscordGuildID: guildID,
		Name:           guildNameValue,
	})
	if err != nil {
		return err
	}

	// Prepare config parameters
	configParams := sqlcdb.UpsertConfigParams{
		DiscordGuildID:    guildID,
		CommandChannelID:  sql.NullString{String: commandChannelID, Valid: commandChannelID != ""},
		OfficerRoleID:     nullStringFromPtr(officerRoleID),
		GuildMemberRoleID: nullStringFromPtr(guildMemberRoleID),
		MercenaryRoleID:   nullStringFromPtr(mercenaryRoleID),
	}

	// Upsert config row
	err = qtx.UpsertConfig(ctx, configParams)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// Helper function to convert *string to sql.NullString
func nullStringFromPtr(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: *s, Valid: true}
}
