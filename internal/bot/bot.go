package bot

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"sync"
	"time"

	"aerobot/internal/config"
	"aerobot/internal/database"
	"aerobot/internal/models"
	"aerobot/internal/transcript"
	"github.com/bwmarrin/discordgo"
)

// Bot manages the Discord ticket bot lifecycle and events
type Bot struct {
	session *discordgo.Session
	cfg     *config.Config
	db      *database.DB
	tg      *transcript.Generator

	// closing tracks channels with a closure in progress to avoid duplicate transcripts.
	closing sync.Map
	// categoryPrompt maps a ticket channel ID to the ID of its category-selection prompt message.
	categoryPrompt sync.Map
	// categoryPickerTokens tracks ephemeral category pickers so they can be removed after use.
	categoryPickerTokens sync.Map
}

// New creates a new Bot instance
func New(cfg *config.Config, db *database.DB, tg *transcript.Generator) (*Bot, error) {
	session, err := discordgo.New("Bot " + cfg.BotToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create discord session: %w", err)
	}

	// Ticket flows are interaction-driven, but transcript generation reads message
	// content through the channel message API, which requires Message Content access.
	session.Identify.Intents = discordgo.IntentsGuilds | discordgo.IntentsMessageContent

	b := &Bot{
		session: session,
		cfg:     cfg,
		db:      db,
		tg:      tg,
	}

	// Register event handlers
	session.AddHandler(b.handleReady)
	session.AddHandler(b.handleInteractionCreate)
	session.AddHandler(b.handleChannelDelete)

	return b, nil
}

// Start opens the Discord WebSocket connection and registers slash commands
func (b *Bot) Start() error {
	if err := b.session.Open(); err != nil {
		return fmt.Errorf("failed to open discord session: %w", err)
	}

	log.Printf("[Bot] Connected to Discord as %s", b.session.State.User.Username)

	// Register application slash commands for the specified guild
	for _, cmd := range ApplicationCommands {
		_, err := b.session.ApplicationCommandCreate(b.session.State.User.ID, b.cfg.GuildID, cmd)
		if err != nil {
			log.Printf("[Warning] Failed to register slash command %q: %v", cmd.Name, err)
		} else {
			log.Printf("[Bot] Registered slash command: /%s", cmd.Name)
		}
	}

	// Reconcile tickets whose channels were removed while the bot was offline.
	go b.reconcileOrphanedTickets()

	return nil
}

// Stop closes the Discord session
func (b *Bot) Stop() error {
	log.Println("[Bot] Shutting down Discord session...")
	return b.session.Close()
}

func (b *Bot) handleReady(s *discordgo.Session, r *discordgo.Ready) {
	log.Printf("[Bot] Logged in as: %s#%s (ID: %s)", r.User.Username, r.User.Discriminator, r.User.ID)
	s.UpdateGameStatus(0, "Managing Support Tickets")
}

// handleChannelDelete closes any ticket whose channel was deleted outside the normal flow.
// Without this, the ticket would remain "active" forever and block the user from opening a new one.
func (b *Bot) handleChannelDelete(s *discordgo.Session, e *discordgo.ChannelDelete) {
	if e == nil || e.Channel == nil || e.Channel.ID == "" {
		return
	}

	ticket, err := b.db.GetTicketByChannelID(e.Channel.ID)
	if err != nil {
		return
	}
	if ticket.Status == models.StatusClosed {
		return
	}

	if err := b.db.UpdateTicketStatus(e.Channel.ID, models.StatusClosed); err != nil {
		log.Printf("[Warning] Failed to mark ticket #%d CLOSED after channel deletion: %v", ticket.ID, err)
	} else {
		log.Printf("[Bot] Marked ticket #%d CLOSED because channel %s was deleted.", ticket.ID, e.Channel.ID)
	}
	b.categoryPrompt.Delete(e.Channel.ID)
	b.closing.Delete(e.Channel.ID)
}

// reconcileOrphanedTickets closes tickets whose channels no longer exist (e.g. deleted while offline).
func (b *Bot) reconcileOrphanedTickets() {
	tickets, err := b.db.GetActiveTickets()
	if err != nil {
		log.Printf("[Warning] Failed to load active tickets for reconciliation: %v", err)
		return
	}

	for _, ticket := range tickets {
		_, err := b.session.Channel(ticket.ChannelID)
		if err == nil {
			continue
		}

		var restErr *discordgo.RESTError
		if errors.As(err, &restErr) && restErr.Message != nil && restErr.Message.Code == discordgo.ErrCodeUnknownChannel {
			if updateErr := b.db.UpdateTicketStatus(ticket.ChannelID, models.StatusClosed); updateErr != nil {
				log.Printf("[Warning] Failed to close orphaned ticket #%d: %v", ticket.ID, updateErr)
			} else {
				log.Printf("[Bot] Closed orphaned ticket #%d (missing channel %s).", ticket.ID, ticket.ChannelID)
			}
			continue
		}

		log.Printf("[Warning] Could not verify channel %s for ticket #%d: %v", ticket.ChannelID, ticket.ID, err)
	}
}

// fetchAllMessages retrieves all messages in a channel in chronological order
func (b *Bot) fetchAllMessages(channelID string) ([]*discordgo.Message, error) {
	var allMessages []*discordgo.Message
	var lastMessageID string
	limit := 100

	for {
		batch, err := b.session.ChannelMessages(channelID, limit, lastMessageID, "", "")
		if err != nil {
			return nil, fmt.Errorf("failed to fetch message batch: %w", err)
		}

		if len(batch) == 0 {
			break
		}

		allMessages = append(allMessages, batch...)
		lastMessageID = batch[len(batch)-1].ID

		if len(batch) < limit {
			break
		}
	}

	// ChannelMessages returns newest first; reverse slice for chronological order
	for i, j := 0, len(allMessages)-1; i < j; i, j = i+1, j-1 {
		allMessages[i], allMessages[j] = allMessages[j], allMessages[i]
	}

	return allMessages, nil
}

// closeTicketAndGenerateTranscript coordinates transcript generation, logging, uploading, and channel deletion
func (b *Bot) closeTicketAndGenerateTranscript(s *discordgo.Session, channelID string, closedBy string) {
	defer b.closing.Delete(channelID)
	b.categoryPrompt.Delete(channelID)

	ticket, err := b.db.GetTicketByChannelID(channelID)
	if err != nil {
		log.Printf("[Error] Failed to find ticket for channel %s: %v", channelID, err)
		s.ChannelMessageSend(channelID, "❌ Error retrieving ticket record from database.")
		return
	}

	if ticket.Status == models.StatusClosed {
		log.Printf("[Bot] Ticket #%d (channel %s) is already closed; skipping duplicate closure.", ticket.ID, channelID)
		return
	}

	// Fetch channel metadata
	channel, err := s.Channel(channelID)
	channelName := fmt.Sprintf("ticket-%d", ticket.ID)
	if err == nil && channel != nil && channel.Name != "" {
		channelName = channel.Name
	}

	// Fetch user info for transcript
	userTag := ticket.UserID
	user, err := s.User(ticket.UserID)
	if err == nil && user != nil {
		if user.Discriminator != "0" && user.Discriminator != "" {
			userTag = fmt.Sprintf("%s#%s", user.Username, user.Discriminator)
		} else {
			userTag = user.Username
		}
	}

	// 1. Fetch all messages chronologically
	messages, err := b.fetchAllMessages(channelID)
	if err != nil {
		log.Printf("[Error] Failed to fetch channel messages for %s: %v", channelID, err)
		s.ChannelMessageSend(channelID, fmt.Sprintf("❌ Failed to fetch channel messages: %v. Channel deletion aborted.", err))
		return
	}

	// 2. Generate and save Markdown transcript to local disk
	closedAt := time.Now().UTC()
	filePath, fileBytes, err := b.tg.GenerateTranscript(ticket, channelName, userTag, messages, closedAt)
	if err != nil {
		log.Printf("[Error] Failed to generate local transcript for ticket %d: %v", ticket.ID, err)
		s.ChannelMessageSend(channelID, fmt.Sprintf("❌ Failed to generate transcript file: %v. Channel deletion aborted.", err))
		return
	}
	log.Printf("[Transcript] Saved local transcript to: %s", filePath)

	// 3. Log transcript to database
	_, err = b.db.LogTranscript(ticket.ID, filePath)
	if err != nil {
		log.Printf("[Warning] Failed to log transcript in database for ticket %d: %v", ticket.ID, err)
	}

	// 4. Upload transcript to the configured transcript channel
	fileName := filepath.Base(filePath)
	transcriptMsg := &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{TranscriptLogEmbed(ticket, closedBy, channelName)},
		Files: []*discordgo.File{
			{
				Name:        fileName,
				ContentType: "text/markdown",
				Reader:      bytes.NewReader(fileBytes),
			},
		},
	}

	_, err = s.ChannelMessageSendComplex(b.cfg.TranscriptChannelID, transcriptMsg)
	if err != nil {
		log.Printf("[Error] Failed to upload transcript to channel %s: %v", b.cfg.TranscriptChannelID, err)
		s.ChannelMessageSend(channelID, fmt.Sprintf("❌ Failed to upload transcript to archive channel (<#%s>): %v. Channel deletion aborted for safety.", b.cfg.TranscriptChannelID, err))
		return
	}

	// 5. Send copy of transcript to the ticket creator via Direct Message (DM)
	b.sendUserClosureDM(s, ticket, channelName, fileName, fileBytes)

	// 6. Update ticket status to CLOSED
	err = b.db.UpdateTicketStatus(channelID, models.StatusClosed)
	if err != nil {
		log.Printf("[Warning] Failed to set status to CLOSED for channel %s: %v", channelID, err)
	}

	// 7. Notify channel and schedule deletion
	s.ChannelMessageSend(channelID, "✅ **Transcript saved and archived successfully.** Deleting channel in 5 seconds...")
	time.Sleep(5 * time.Second)

	_, err = s.ChannelDelete(channelID)
	if err != nil {
		log.Printf("[Error] Failed to delete channel %s: %v", channelID, err)
	} else {
		log.Printf("[Bot] Ticket #%d (channel %s) closed and deleted successfully.", ticket.ID, channelID)
	}
}

// sendUserClosureDM sends the Markdown transcript directly to the ticket creator via DM
func (b *Bot) sendUserClosureDM(s *discordgo.Session, ticket *models.Ticket, channelName, fileName string, fileBytes []byte) {
	dmChannel, err := s.UserChannelCreate(ticket.UserID)
	if err != nil {
		log.Printf("[Warning] Failed to open DM channel with ticket creator %s: %v (user may have DMs disabled)", ticket.UserID, err)
		return
	}

	msgContent := fmt.Sprintf(
		"Hello! Your support ticket **#%d** (`#%s`) has been closed.\n\n"+
			"Attached to this message is the `.md` file (**%s**), which is the **full chat transcript** of your support ticket for your personal records.",
		ticket.ID, channelName, fileName,
	)

	userMsg := &discordgo.MessageSend{
		Content: msgContent,
		Embeds:  []*discordgo.MessageEmbed{UserTranscriptClosedEmbed(ticket, channelName, fileName)},
		Files: []*discordgo.File{
			{
				Name:        fileName,
				ContentType: "text/markdown",
				Reader:      bytes.NewReader(fileBytes),
			},
		},
	}

	_, err = s.ChannelMessageSendComplex(dmChannel.ID, userMsg)
	if err != nil {
		log.Printf("[Warning] Failed to send transcript DM to user %s: %v (user may have DMs closed or blocked bot)", ticket.UserID, err)
	} else {
		log.Printf("[Bot] Successfully dispatched ticket #%d transcript DM to user %s", ticket.ID, ticket.UserID)
	}
}
