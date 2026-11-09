package transcript

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"aerobot/internal/models"
	"github.com/bwmarrin/discordgo"
)

// Generator produces Markdown transcripts from tickets and message histories
type Generator struct {
	folderPath string
}

// New creates a new transcript Generator instance
func New(folderPath string) (*Generator, error) {
	if folderPath == "" {
		folderPath = "./transcripts"
	}
	if err := os.MkdirAll(folderPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create transcript directory: %w", err)
	}
	return &Generator{folderPath: folderPath}, nil
}

// GenerateTranscript formats the messages into Markdown and writes the file to disk
func (g *Generator) GenerateTranscript(
	ticket *models.Ticket,
	channelName string,
	userTag string,
	messages []*discordgo.Message,
	closedAt time.Time,
) (string, []byte, error) {
	if channelName == "" {
		channelName = fmt.Sprintf("ticket-%d", ticket.ID)
	}
	if userTag == "" {
		userTag = ticket.UserID
	}

	timeFormat := "2006-01-02 15:04:05"
	closedAtStr := closedAt.UTC().Format(timeFormat)

	nickname := ticket.IngameNickname
	if nickname == "" {
		nickname = "N/A"
	}
	category := ticket.Category
	if category == "" {
		category = "N/A"
	}
	categoryDetails := ticket.CategoryDetails
	additionalInfo := ticket.AdditionalInfo
	if categoryDetails == "" && (ticket.Category == string(models.CategoryGriefing) || ticket.Category == string(models.CategoryShip)) {
		categoryDetails = additionalInfo
		additionalInfo = ""
	}
	if additionalInfo == "" {
		additionalInfo = "N/A"
	}
	problemDesc := ticket.ProblemDescription
	if problemDesc == "" {
		problemDesc = "N/A"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# TICKET TRANSCRIPT: #%s\n", channelName))
	sb.WriteString(fmt.Sprintf("- **Ticket ID:** %d\n", ticket.ID))
	sb.WriteString(fmt.Sprintf("- **User:** %s (%s)\n", userTag, ticket.UserID))
	sb.WriteString(fmt.Sprintf("- **In-Game Nickname:** %s\n", nickname))
	sb.WriteString(fmt.Sprintf("- **Category:** %s\n", category))
	if label := models.CategoryDetailLabel(ticket.Category); label != "" {
		if categoryDetails == "" {
			categoryDetails = "N/A"
		}
		sb.WriteString(fmt.Sprintf("- **%s:** %s\n", label, categoryDetails))
	}
	sb.WriteString(fmt.Sprintf("- **Additional Info:** %s\n", additionalInfo))
	sb.WriteString(fmt.Sprintf("- **%s:** %s\n", models.CategoryProblemLabel(ticket.Category), problemDesc))
	sb.WriteString(fmt.Sprintf("- **Closed At:** %s (UTC)\n\n", closedAtStr))
	sb.WriteString("---\n")
	sb.WriteString("## MESSAGE HISTORY\n\n")

	// Messages are formatted chronologically
	for _, msg := range messages {
		if msg == nil {
			continue
		}

		timestamp := msg.Timestamp.UTC().Format(timeFormat)
		authorName := "Unknown"
		if msg.Author != nil {
			if msg.Author.Discriminator != "0" && msg.Author.Discriminator != "" {
				authorName = fmt.Sprintf("%s#%s", msg.Author.Username, msg.Author.Discriminator)
			} else {
				authorName = msg.Author.Username
			}
		}

		content := msg.Content
		// Append attachments if any
		if len(msg.Attachments) > 0 {
			var attachList []string
			for _, att := range msg.Attachments {
				attachList = append(attachList, fmt.Sprintf("[Attachment: %s (%s)]", att.Filename, att.URL))
			}
			if content != "" {
				content += " " + strings.Join(attachList, " ")
			} else {
				content = strings.Join(attachList, " ")
			}
		}

		// Append embeds summary if message has no content but has embeds
		if content == "" && len(msg.Embeds) > 0 {
			var embedTitles []string
			for _, emb := range msg.Embeds {
				if emb.Title != "" {
					embedTitles = append(embedTitles, fmt.Sprintf("[Embed: %s]", emb.Title))
				} else if emb.Description != "" {
					descPreview := emb.Description
					if len(descPreview) > 60 {
						descPreview = descPreview[:60] + "..."
					}
					embedTitles = append(embedTitles, fmt.Sprintf("[Embed: %s]", descPreview))
				}
			}
			content = strings.Join(embedTitles, " ")
		}

		if content == "" {
			content = "(empty message)"
		}

		sb.WriteString(fmt.Sprintf("[%s] %s: %s\n", timestamp, authorName, content))
	}

	contentBytes := []byte(sb.String())

	// Build safe filename: ticket-<zero-padded id>-<sanitized_user>.md
	safeUser := sanitizeFilename(userTag)
	filename := fmt.Sprintf("ticket-%04d-%s.md", ticket.ID, safeUser)
	fullPath := filepath.Join(g.folderPath, filename)

	if err := os.WriteFile(fullPath, contentBytes, 0644); err != nil {
		return "", nil, fmt.Errorf("failed to save transcript file %q: %w", fullPath, err)
	}

	return fullPath, contentBytes, nil
}

var nonAlphanumericRegex = regexp.MustCompile(`[^a-zA-Z0-9_\-]+`)

func sanitizeFilename(s string) string {
	s = strings.ReplaceAll(s, "#", "_")
	s = nonAlphanumericRegex.ReplaceAllString(s, "_")
	s = strings.Trim(s, "_")
	if s == "" {
		return "user"
	}
	if len(s) > 30 {
		return s[:30]
	}
	return s
}
