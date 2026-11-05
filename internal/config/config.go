package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

// Config holds the application configuration parameters
type Config struct {
	BotToken             string   `json:"bot_token"`
	GuildID              string   `json:"guild_id"`
	CategoryID           string   `json:"category_id"`
	TranscriptChannelID  string   `json:"transcript_channel_id"`
	TranscriptFolderPath string   `json:"transcript_folder_path"`
	StaffRoleIDs         []string `json:"staff_role_ids"`
	DatabasePath         string   `json:"database_path"`
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		TranscriptFolderPath: "./transcripts",
		DatabasePath:         "./aerobot.db",
		StaffRoleIDs:         make([]string, 0),
	}
}

// Load loads configuration from a JSON file and environment variables
func Load(path string) (*Config, error) {
	cfg := DefaultConfig()

	// Try reading file if specified or if default exists
	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			if !os.IsNotExist(err) {
				return nil, fmt.Errorf("failed to read config file %q: %w", path, err)
			}
		} else if strings.TrimSpace(string(data)) != "" {
			if err := json.Unmarshal(data, cfg); err != nil {
				return nil, fmt.Errorf("failed to parse config JSON from %q: %w", path, err)
			}
		}
	} else if data, err := os.ReadFile("config.json"); err == nil && strings.TrimSpace(string(data)) != "" {
		if err := json.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("failed to parse config.json: %w", err)
		}
	}

	// Environment variable overrides
	if token := getEnv("BOT_TOKEN", "DISCORD_BOT_TOKEN"); token != "" {
		cfg.BotToken = token
	}
	if guild := getEnv("GUILD_ID", "DISCORD_GUILD_ID"); guild != "" {
		cfg.GuildID = guild
	}
	if cat := getEnv("CATEGORY_ID", "DISCORD_CATEGORY_ID"); cat != "" {
		cfg.CategoryID = cat
	}
	if tChan := getEnv("TRANSCRIPT_CHANNEL_ID", "DISCORD_TRANSCRIPT_CHANNEL_ID"); tChan != "" {
		cfg.TranscriptChannelID = tChan
	}
	if tFolder := getEnv("TRANSCRIPT_FOLDER_PATH"); tFolder != "" {
		cfg.TranscriptFolderPath = tFolder
	}
	if dbPath := getEnv("DATABASE_PATH", "DB_PATH"); dbPath != "" {
		cfg.DatabasePath = dbPath
	}
	if staffRoles := getEnv("STAFF_ROLE_IDS"); staffRoles != "" {
		roles := strings.Split(staffRoles, ",")
		var cleaned []string
		for _, r := range roles {
			trimmed := strings.TrimSpace(r)
			if trimmed != "" {
				cleaned = append(cleaned, trimmed)
			}
		}
		if len(cleaned) > 0 {
			cfg.StaffRoleIDs = cleaned
		}
	}

	// Normalize paths
	cfg.TranscriptFolderPath = strings.TrimRight(cfg.TranscriptFolderPath, "/\\")

	return cfg, nil
}

// Validate ensures all required configuration parameters are present
func (c *Config) Validate() error {
	if strings.TrimSpace(c.BotToken) == "" {
		return errors.New("missing bot_token: Discord Bot Token is required")
	}
	if isPlaceholderToken(c.BotToken) {
		return errors.New("invalid bot_token: configuration still contains the example placeholder, set your real Discord Bot Token")
	}
	if strings.TrimSpace(c.GuildID) == "" {
		return errors.New("missing guild_id: Discord Server/Guild ID is required")
	}
	if isPlaceholderSnowflake(c.GuildID) {
		return errors.New("invalid guild_id: configuration still contains an example placeholder, set your real Discord Server/Guild ID")
	}
	if strings.TrimSpace(c.CategoryID) == "" {
		return errors.New("missing category_id: Category ID for ticket channels is required")
	}
	if isPlaceholderSnowflake(c.CategoryID) {
		return errors.New("invalid category_id: configuration still contains an example placeholder, set the real ticket category ID")
	}
	if strings.TrimSpace(c.TranscriptChannelID) == "" {
		return errors.New("missing transcript_channel_id: Transcript Channel ID is required")
	}
	if isPlaceholderSnowflake(c.TranscriptChannelID) {
		return errors.New("invalid transcript_channel_id: configuration still contains an example placeholder, set the real transcript channel ID")
	}
	for _, roleID := range c.StaffRoleIDs {
		if isPlaceholderSnowflake(roleID) {
			return errors.New("invalid staff_role_ids: configuration still contains example placeholder role IDs")
		}
	}
	if strings.TrimSpace(c.TranscriptFolderPath) == "" {
		c.TranscriptFolderPath = "./transcripts"
	}
	if strings.TrimSpace(c.DatabasePath) == "" {
		c.DatabasePath = "./aerobot.db"
	}
	return nil
}

// isPlaceholderToken reports whether the token is still the example placeholder.
func isPlaceholderToken(token string) bool {
	upper := strings.ToUpper(strings.TrimSpace(token))
	return strings.HasPrefix(upper, "YOUR_") || strings.Contains(upper, "TOKEN_HERE")
}

// isPlaceholderSnowflake reports whether an ID is one of the example IDs shipped in config.example.json.
func isPlaceholderSnowflake(id string) bool {
	switch strings.TrimSpace(id) {
	case "123456789012345678", "234567890123456789":
		return true
	default:
		return false
	}
}

func getEnv(keys ...string) string {
	for _, key := range keys {
		if val, exists := os.LookupEnv(key); exists && strings.TrimSpace(val) != "" {
			return strings.TrimSpace(val)
		}
	}
	return ""
}
