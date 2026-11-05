package transcript_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"aerobot/internal/models"
	"aerobot/internal/transcript"
	"github.com/bwmarrin/discordgo"
)

func TestGenerateTranscript(t *testing.T) {
	tempDir := t.TempDir()
	tg, err := transcript.New(tempDir)
	if err != nil {
		t.Fatalf("failed to create transcript generator: %v", err)
	}

	ticket := &models.Ticket{
		ID:                 42,
		ChannelID:          "chan-4242",
		UserID:             "user-100",
		Status:             models.StatusResolved,
		Category:           string(models.CategoryGriefing),
		IngameNickname:     "ShadowNinja",
		CategoryDetails:    "X: 120, Y: 64, Z: -450",
		AdditionalInfo:     "Seen near the station",
		ProblemDescription: "Base was raided during peace period",
		CreatedAt:          time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC),
	}

	messages := []*discordgo.Message{
		{
			ID:        "msg-1",
			ChannelID: "chan-4242",
			Content:   "Hello! My nickname is ShadowNinja",
			Author: &discordgo.User{
				ID:            "user-100",
				Username:      "ShadowUser",
				Discriminator: "1234",
			},
			Timestamp: time.Date(2026, 9, 11, 12, 1, 0, 0, time.UTC),
		},
		{
			ID:        "msg-2",
			ChannelID: "chan-4242",
			Content:   "Thank you, ShadowNinja! What is the ticket about?",
			Author: &discordgo.User{
				ID:            "bot-1",
				Username:      "AeroBot",
				Discriminator: "0000",
				Bot:           true,
			},
			Timestamp: time.Date(2026, 9, 11, 12, 1, 5, 0, time.UTC),
		},
		{
			ID:        "msg-3",
			ChannelID: "chan-4242",
			Content:   "Here are screenshots",
			Author: &discordgo.User{
				ID:            "user-100",
				Username:      "ShadowUser",
				Discriminator: "1234",
			},
			Timestamp: time.Date(2026, 9, 11, 12, 2, 0, 0, time.UTC),
			Attachments: []*discordgo.MessageAttachment{
				{
					Filename: "proof.png",
					URL:      "https://cdn.discordapp.com/attachments/proof.png",
				},
			},
		},
	}

	closedAt := time.Date(2026, 9, 11, 12, 10, 0, 0, time.UTC)
	filePath, fileBytes, err := tg.GenerateTranscript(ticket, "ticket-shadowninja", "ShadowUser#1234", messages, closedAt)
	if err != nil {
		t.Fatalf("failed to generate transcript: %v", err)
	}
	if expected := "ticket-0042-ShadowUser_1234.md"; filepath.Base(filePath) != expected {
		t.Errorf("expected transcript filename %s, got %s", expected, filepath.Base(filePath))
	}

	// Verify file exists on disk
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read generated transcript file %s: %v", filePath, err)
	}

	strContent := string(content)

	// Validate expected structure from prompt specification:
	// # TICKET TRANSCRIPT: #ticket-shadowninja
	// - **Ticket ID:** 42
	// - **User:** ShadowUser#1234 (user-100)
	// - **In-Game Nickname:** ShadowNinja
	// - **Category:** Griefing / Stealing
	// - **Coordinates:** X: 120, Y: 64, Z: -450
	// - **Additional Info:** Seen near the station
	// - **Closed At:** 2026-09-11 12:10:00 (UTC)
	if !strings.Contains(strContent, "# TICKET TRANSCRIPT: #ticket-shadowninja") {
		t.Errorf("missing header in transcript")
	}
	if !strings.Contains(strContent, "- **Ticket ID:** 42") {
		t.Errorf("missing ticket ID in transcript")
	}
	if !strings.Contains(strContent, "- **In-Game Nickname:** ShadowNinja") {
		t.Errorf("missing nickname in transcript")
	}
	if !strings.Contains(strContent, "- **Category:** Griefing / Stealing") {
		t.Errorf("missing category in transcript")
	}
	if !strings.Contains(strContent, "- **Coordinates:** X: 120, Y: 64, Z: -450") {
		t.Errorf("missing category details in transcript")
	}
	if !strings.Contains(strContent, "- **Additional Info:** Seen near the station") {
		t.Errorf("missing additional info in transcript")
	}
	if !strings.Contains(strContent, "## MESSAGE HISTORY") {
		t.Errorf("missing MESSAGE HISTORY section")
	}
	if !strings.Contains(strContent, "[2026-09-11 12:01:00] ShadowUser#1234: Hello! My nickname is ShadowNinja") {
		t.Errorf("missing message 1 in transcript: %s", strContent)
	}
	if !strings.Contains(strContent, "[2026-09-11 12:01:05] AeroBot#0000: Thank you, ShadowNinja! What is the ticket about?") {
		t.Errorf("missing bot message in transcript: %s", strContent)
	}
	if !strings.Contains(strContent, "[Attachment: proof.png (https://cdn.discordapp.com/attachments/proof.png)]") {
		t.Errorf("missing attachment link in transcript: %s", strContent)
	}

	if len(fileBytes) == 0 {
		t.Errorf("fileBytes should not be empty")
	}
}
