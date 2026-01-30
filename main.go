package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"

	"PanickedBot/internal"
	"PanickedBot/internal/commands"
	"PanickedBot/internal/db"
)

// deregisterAllCommands removes all globally registered application commands
func deregisterAllCommands(dg *discordgo.Session) error {
	appID := dg.State.User.ID
	
	// Get all existing global commands
	existingCommands, err := dg.ApplicationCommands(appID, "")
	if err != nil {
		return err
	}

	log.Printf("Found %d registered commands to deregister", len(existingCommands))

	// Track any errors but continue attempting to delete all commands
	var errors []string

	// Delete each command
	for _, cmd := range existingCommands {
		err := dg.ApplicationCommandDelete(appID, "", cmd.ID)
		if err != nil {
			errMsg := fmt.Sprintf("Failed to delete command /%s: %v", cmd.Name, err)
			log.Println(errMsg)
			errors = append(errors, errMsg)
			continue
		}
		log.Printf("Deregistered global /%s", cmd.Name)
	}

	// If any errors occurred, return them
	if len(errors) > 0 {
		errorMsg := strings.Join(errors, "\n")
		return fmt.Errorf("failed to deregister some commands:\n%s", errorMsg)
	}

	log.Printf("Successfully deregistered all commands")
	return nil
}

func main() {
	// Parse command-line flags
	deregister := flag.Bool("deregister", false, "Deregister all Discord commands and exit")
	flag.Parse()

	cfg, err := internal.LoadConfigFromEnv()
	if err != nil {
		log.Fatal(err)
	}

	// Connect to Discord
	dg, err := discordgo.New("Bot " + cfg.DiscordToken)
	if err != nil {
		log.Fatalf("discord session: %v", err)
	}
	dg.Identify.Intents = discordgo.IntentsGuilds

	if err := dg.Open(); err != nil {
		log.Fatalf("discord open: %v", err)
	}
	defer dg.Close()

	// If deregister flag is set, deregister commands and exit
	if *deregister {
		if err := deregisterAllCommands(dg); err != nil {
			log.Fatalf("Failed to deregister commands: %v", err)
		}
		log.Println("Deregistration complete. Exiting.")
		return
	}

	// Normal startup - connect to database
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

	appID := dg.State.User.ID

	cmds := commands.GetCommands()

	dg.AddHandler(commands.CreateInteractionHandler(database))

	registered := make([]*discordgo.ApplicationCommand, 0, len(cmds))
	for _, cmd := range cmds {
		rc, err := dg.ApplicationCommandCreate(appID, "", cmd)
		if err != nil {
			log.Fatalf("command create (%s): %v", cmd.Name, err)
		}
		registered = append(registered, rc)
		log.Printf("registered global /%s", cmd.Name)
	}

	if err := internal.EnsureGuildRows(database, dg.State.Guilds); err != nil {
		log.Printf("bootstrap guild rows warning: %v", err)
	}

	log.Printf("bot ready (app=%s)", appID)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()

	_ = registered
}
