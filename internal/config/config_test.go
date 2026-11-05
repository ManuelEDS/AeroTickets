package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"aerobot/internal/config"
)

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name      string
		cfg       config.Config
		wantError bool
	}{
		{
			name: "valid config",
			cfg: config.Config{
				BotToken:            "test-token",
				GuildID:             "123",
				CategoryID:          "456",
				TranscriptChannelID: "789",
				DatabasePath:        ":memory:",
			},
			wantError: false,
		},
		{
			name: "missing bot token",
			cfg: config.Config{
				BotToken:            "",
				GuildID:             "123",
				CategoryID:          "456",
				TranscriptChannelID: "789",
			},
			wantError: true,
		},
		{
			name: "missing guild id",
			cfg: config.Config{
				BotToken:            "token",
				GuildID:             "",
				CategoryID:          "456",
				TranscriptChannelID: "789",
			},
			wantError: true,
		},
		{
			name: "missing category id",
			cfg: config.Config{
				BotToken:            "token",
				GuildID:             "123",
				CategoryID:          "",
				TranscriptChannelID: "789",
			},
			wantError: true,
		},
		{
			name: "missing transcript channel id",
			cfg: config.Config{
				BotToken:            "token",
				GuildID:             "123",
				CategoryID:          "456",
				TranscriptChannelID: "",
			},
			wantError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.cfg.Validate()
			if (err != nil) != tc.wantError {
				t.Fatalf("expected error: %v, got: %v", tc.wantError, err)
			}
		})
	}
}

func TestLoadEmptyConfigFileDoesNotFail(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.json")
	if err := os.WriteFile(configFile, []byte{}, 0644); err != nil {
		t.Fatalf("failed to write empty config file: %v", err)
	}

	cfg, err := config.Load(configFile)
	if err != nil {
		t.Fatalf("expected empty config file to be tolerated, got: %v", err)
	}
	if cfg.DatabasePath != "./aerobot.db" {
		t.Errorf("expected default database path, got %q", cfg.DatabasePath)
	}
}

func TestValidateRejectsPlaceholders(t *testing.T) {
	cfg := config.Config{
		BotToken:            "YOUR_DISCORD_BOT_TOKEN_HERE",
		GuildID:             "123456789012345678",
		CategoryID:          "987654321098765432",
		TranscriptChannelID: "876543210987654321",
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected placeholder token to be rejected")
	}
}

func TestLoadConfigFileAndEnv(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.json")
	jsonContent := `{
		"bot_token": "file-token",
		"guild_id": "file-guild",
		"category_id": "file-cat",
		"transcript_channel_id": "file-transcript-chan",
		"transcript_folder_path": "./custom-transcripts",
		"staff_role_ids": ["role1", "role2"]
	}`

	if err := os.WriteFile(configFile, []byte(jsonContent), 0644); err != nil {
		t.Fatalf("failed to write test config file: %v", err)
	}

	cfg, err := config.Load(configFile)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if cfg.BotToken != "file-token" {
		t.Errorf("expected bot_token 'file-token', got '%s'", cfg.BotToken)
	}
	if cfg.GuildID != "file-guild" {
		t.Errorf("expected guild_id 'file-guild', got '%s'", cfg.GuildID)
	}
	if len(cfg.StaffRoleIDs) != 2 || cfg.StaffRoleIDs[0] != "role1" {
		t.Errorf("unexpected staff roles: %v", cfg.StaffRoleIDs)
	}

	// Test ENV override
	os.Setenv("BOT_TOKEN", "env-token-override")
	defer os.Unsetenv("BOT_TOKEN")

	cfgEnv, err := config.Load(configFile)
	if err != nil {
		t.Fatalf("failed to load config with env: %v", err)
	}
	if cfgEnv.BotToken != "env-token-override" {
		t.Errorf("expected env override 'env-token-override', got '%s'", cfgEnv.BotToken)
	}
}
