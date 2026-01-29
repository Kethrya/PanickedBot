package internal

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/jmoiron/sqlx"
)

// EnsureGuildRows ensures that guild rows exist in the database
func EnsureGuildRows(dbx *sqlx.DB, guildStates []*discordgo.Guild) error {
	if len(guildStates) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tx, err := dbx.BeginTxx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	// Insert guild rows if missing; do not overwrite name.
	// MariaDB: INSERT IGNORE is easiest here.
	stmt := `INSERT IGNORE INTO guilds (discord_guild_id, name) VALUES (?, ?)`
	for _, g := range guildStates {
		if g == nil || g.ID == "" {
			continue
		}
		if _, err := tx.ExecContext(ctx, stmt, g.ID, g.Name); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// NullIfEmpty returns nil if the string is empty, otherwise returns the string
func NullIfEmpty(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}

// NullIfEmptyPtr returns nil if the string is empty, otherwise returns a pointer to the string
func NullIfEmptyPtr(s string) *string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return &s
}
