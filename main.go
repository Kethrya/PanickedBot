package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"

	"PanickedBot/internal/db"
)

func main() {
	cfg, err := loadConfigFromEnv()
	if err != nil {
		log.Fatal(err)
	}

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

	if err := database.PingContext(context.Background()); err != nil {
		log.Fatalf("db ping: %v", err)
	}

	dg, err := discordgo.New("Bot " + cfg.DiscordToken)
	if err != nil {
		log.Fatalf("discord session: %v", err)
	}
	dg.Identify.Intents = discordgo.IntentsGuilds

	commands := getCommands()

	dg.AddHandler(createInteractionHandler(database))

	if err := dg.Open(); err != nil {
		log.Fatalf("discord open: %v", err)
	}
	defer dg.Close()

	appID := dg.State.User.ID

	registered := make([]*discordgo.ApplicationCommand, 0, len(commands))
	for _, cmd := range commands {
		rc, err := dg.ApplicationCommandCreate(appID, "", cmd)
		if err != nil {
			log.Fatalf("command create (%s): %v", cmd.Name, err)
		}
		registered = append(registered, rc)
		log.Printf("registered global /%s", cmd.Name)
	}

	if err := ensureGuildRows(database, dg.State.Guilds); err != nil {
		log.Printf("bootstrap guild rows warning: %v", err)
	}

	log.Printf("bot ready (app=%s)", appID)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()

	_ = registered
}
