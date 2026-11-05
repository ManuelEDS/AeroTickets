package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"aerobot/internal/bot"
	"aerobot/internal/config"
	"aerobot/internal/database"
	"aerobot/internal/transcript"
)

const Banner = `
  /\  ___ _ __ ___ | __ )  ___ | |_ 
 / _ \ / _ \ '__/ _ \|  _ \ / _ \| __|
/ ___ \  __/ | | (_) | |_) | (_) | |_ 
/_/   \_\___|_|  \___/|____/ \___/ \__|
  Discord Support Ticket Management Bot
`

func main() {
	configPath := flag.String("config", "config.json", "Path to configuration file")
	flag.Parse()

	fmt.Print(Banner)
	log.Println("[Init] Starting AeroBot Support Ticket System...")

	// 1. Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("[Fatal] Failed to load configuration: %v", err)
	}

	// 2. Validate configuration
	if err := cfg.Validate(); err != nil {
		log.Fatalf("[Fatal] Configuration validation error: %v", err)
	}
	log.Printf("[Config] Loaded configuration. Guild: %s, Category: %s, Transcript Channel: %s",
		cfg.GuildID, cfg.CategoryID, cfg.TranscriptChannelID)

	// 3. Initialize SQLite Database
	db, err := database.New(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("[Fatal] Failed to initialize SQLite database (%s): %v", cfg.DatabasePath, err)
	}
	defer db.Close()
	log.Printf("[Database] Connected and migrated SQLite database at: %s", cfg.DatabasePath)

	// 4. Initialize Transcript Generator
	tg, err := transcript.New(cfg.TranscriptFolderPath)
	if err != nil {
		log.Fatalf("[Fatal] Failed to initialize transcript generator: %v", err)
	}
	log.Printf("[Transcript] Transcript storage initialized at: %s", cfg.TranscriptFolderPath)

	// 5. Initialize Discord Bot
	discordBot, err := bot.New(cfg, db, tg)
	if err != nil {
		log.Fatalf("[Fatal] Failed to initialize Discord bot: %v", err)
	}

	// 6. Start Discord Connection
	if err := discordBot.Start(); err != nil {
		log.Fatalf("[Fatal] Failed to start Discord bot: %v", err)
	}

	log.Println("[Ready] AeroBot is running! Press Ctrl+C or send SIGINT/SIGTERM to exit.")

	// 7. Wait for termination signal for graceful shutdown
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-stopChan

	log.Println("[Shutdown] Received interrupt signal. Commencing graceful shutdown...")
	discordBot.Stop()
	log.Println("[Shutdown] AeroBot stopped cleanly. Goodbye!")
}
