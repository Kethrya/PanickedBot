package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"

	"PanickedBot/internal/db"
)

func main() {
	// ---- config from env ----
	cfg, err := loadConfigFromEnv()
	if err != nil {
		log.Fatal(err)
	}

	// ---- db ----
	database, err := db.Open(db.Config{
		DSN:             cfg.DatabaseDSN,
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: 30 * time.Minute,
	})
	if err != nil {
		log.Fatalf("db open: %v", err)
	}
	defer database.Close()

	// Optional: verify DB connectivity early
	if err := database.PingContext(context.Background()); err != nil {
		log.Fatalf("db ping: %v", err)
	}

	// ---- discord ----
	dg, err := discordgo.New("Bot " + cfg.DiscordToken)
	if err != nil {
		log.Fatalf("discord session: %v", err)
	}
	dg.Identify.Intents = discordgo.IntentsGuilds

	// Router: map command name -> handler
	handlers := map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"ping": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			respondText(s, i, "pong")
		},
	}

	dg.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if i.Type != discordgo.InteractionApplicationCommand {
			return
		}
		name := i.ApplicationCommandData().Name

		h, ok := handlers[name]
		if !ok {
			respondText(s, i, "Unknown command.")
			return
		}

		// Example guard: if you later want role gating for uploads/config
		// if name == "warscores" && !hasAnyRole(i, cfg.AllowedRoleIDs) { ... }

		h(s, i)
	})

	if err := dg.Open(); err != nil {
		log.Fatalf("discord open: %v", err)
	}
	defer dg.Close()

	// Register slash commands (guild-scoped for fast iteration)
	appID := dg.State.User.ID

	commands := []*discordgo.ApplicationCommand{
		{
			Name:        "ping",
			Description: "health check",
		},
		// Add more here later: setup, roster, warscores, leaderboard...
	}

	registered := make([]*discordgo.ApplicationCommand, 0, len(commands))
	for _, cmd := range commands {
		rc, err := dg.ApplicationCommandCreate(appID, cfg.DevGuildID, cmd)
		if err != nil {
			log.Fatalf("command create (%s): %v", cmd.Name, err)
		}
		registered = append(registered, rc)
		log.Printf("registered /%s", cmd.Name)
	}

	log.Printf("bot ready (app=%s, dev_guild=%s)", appID, cfg.DevGuildID)

	// ---- graceful shutdown ----
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()

	// Delete dev commands on exit so you don’t accumulate junk
	for _, cmd := range registered {
		_ = dg.ApplicationCommandDelete(appID, cfg.DevGuildID, cmd.ID)
	}
}

// -------- helpers --------

type config struct {
	DiscordToken string
	DevGuildID   string
	DatabaseDSN  string

	// Optional: comma-separated list in env, used later for role gates
	AllowedRoleIDs []string
}

func loadConfigFromEnv() (config, error) {
	get := func(key string) string { return strings.TrimSpace(os.Getenv(key)) }

	c := config{
		DiscordToken: get("DISCORD_BOT_TOKEN"),
		DevGuildID:   get("DISCORD_DEV_GUILD_ID"),
		DatabaseDSN:  get("DATABASE_DSN"),
	}

	if c.DiscordToken == "" {
		return c, errors.New("DISCORD_BOT_TOKEN is not set")
	}
	if c.DevGuildID == "" {
		return c, errors.New("DISCORD_DEV_GUILD_ID is not set (register commands in a dev server)")
	}
	if c.DatabaseDSN == "" {
		return c, errors.New("DATABASE_DSN is not set")
	}

	if v := get("ALLOWED_ROLE_IDS"); v != "" {
		parts := strings.Split(v, ",")
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				c.AllowedRoleIDs = append(c.AllowedRoleIDs, p)
			}
		}
	}

	return c, nil
}

func respondText(s *discordgo.Session, i *discordgo.InteractionCreate, msg string) {
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: msg,
		},
	})
}

// hasAnyRole checks roles on the invoking member.
// (Works for guild interactions; DMs won’t have Member populated.)
func hasAnyRole(i *discordgo.InteractionCreate, allowed []string) bool {
	if len(allowed) == 0 {
		return false
	}
	if i.Member == nil {
		return false
	}
	roleSet := make(map[string]struct{}, len(i.Member.Roles))
	for _, r := range i.Member.Roles {
		roleSet[r] = struct{}{}
	}
	for _, a := range allowed {
		if _, ok := roleSet[a]; ok {
			return true
		}
	}
	return false
}
