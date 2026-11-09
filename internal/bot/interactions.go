package bot

import (
	"errors"
	"fmt"
	"log"
	"regexp"
	"strings"

	"aerobot/internal/database"
	"aerobot/internal/models"
	"github.com/bwmarrin/discordgo"
)

type interactionRef struct {
	appID string
	token string
}

const maxActiveTicketsPerUser = 3

var coordinateNumberPattern = regexp.MustCompile(`-?\d+(?:\.\d+)?`)

func sanitizeCoordinates(value string) string {
	return strings.Join(coordinateNumberPattern.FindAllString(value, -1), " ")
}

func categoryPickerKey(userID, channelID string) string {
	return userID + ":" + channelID
}

// handleInteractionCreate routes all incoming interactions
func (b *Bot) handleInteractionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		b.handleSlashCommand(s, i)

	case discordgo.InteractionMessageComponent:
		b.handleComponentInteraction(s, i)

	case discordgo.InteractionModalSubmit:
		b.handleModalSubmit(s, i)
	}
}

// handleComponentInteraction routes button interactions
func (b *Bot) handleComponentInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.MessageComponentData().CustomID

	switch customID {
	case IDTicketCreate:
		b.handleCreateTicketButton(s, i)

	case IDCategoryGriefing, IDCategoryShip, IDCategoryBug, IDCategoryCrash, IDCategoryAppeal, IDCategoryOther:
		b.handleCategoryButton(s, i, customID)

	case IDTicketCloseRequest:
		b.handleCloseRequestButton(s, i)

	case IDTicketCloseConfirm:
		b.handleCloseConfirmButton(s, i)

	case IDTicketCloseCancel:
		b.handleCloseCancelButton(s, i)

	case IDTicketTeleportCopy:
		b.handleTeleportCommandButton(s, i)
	}
}

func (b *Bot) handleTeleportCommandButton(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if !b.isStaffInteraction(i) {
		respondEphemeral(s, i, "Only configured ticket staff or members with Manage Channels/Administrator can use this button.")
		return
	}

	ticket, err := b.db.GetTicketByChannelID(i.ChannelID)
	if err != nil {
		log.Printf("[Error] Failed to load ticket for teleport command in channel %s: %v", i.ChannelID, err)
		respondEphemeral(s, i, "❌ Could not load this ticket. Please try again.")
		return
	}
	if ticket.Category != string(models.CategoryGriefing) {
		respondEphemeral(s, i, "⚠️ Teleport commands are only available for Griefing tickets.")
		return
	}
	coordinates := sanitizeCoordinates(ticket.CategoryDetails)
	if coordinates == "" {
		respondEphemeral(s, i, "⚠️ This ticket does not contain valid coordinates.")
		return
	}
	respondEphemeral(s, i, fmt.Sprintf("Copy this Minecraft command:\n```\n/tp %s\n```", coordinates))
}

// handleCreateTicketButton keeps old panels working by showing the category buttons ephemerally.
func (b *Bot) handleCreateTicketButton(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID, _ := interactionUser(i)
	if userID == "" {
		respondEphemeral(s, i, "⚠️ Could not identify your Discord account. Please try again.")
		return
	}

	activeTickets, err := b.db.CountActiveTicketsForUser(userID)
	if err != nil {
		log.Printf("[Error] Failed to check active ticket for user %s: %v", userID, err)
		respondEphemeral(s, i, "❌ Could not check your existing tickets. Please try again.")
		return
	}
	if activeTickets >= maxActiveTicketsPerUser {
		respondEphemeral(s, i, fmt.Sprintf("⚠️ You already have %d active tickets. The limit is %d; close one before creating another.", activeTickets, maxActiveTicketsPerUser))
		return
	}

	if !respondEphemeralWithComponents(s, i, "📂 Choose the category that matches your issue:", CategorySelectionComponents()) {
		return
	}
	b.categoryPickerTokens.Store(categoryPickerKey(userID, i.ChannelID), interactionRef{
		appID: i.AppID,
		token: i.Token,
	})
}

func interactionUser(i *discordgo.InteractionCreate) (string, string) {
	if i.Member != nil && i.Member.User != nil {
		return i.Member.User.ID, i.Member.User.Username
	}
	if i.User != nil {
		return i.User.ID, i.User.Username
	}
	return "", ""
}

func respondEphemeral(s *discordgo.Session, i *discordgo.InteractionCreate, content string) {
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	}); err != nil {
		log.Printf("[Error] Failed to send interaction response: %v", err)
	}
}

func respondEphemeralWithComponents(s *discordgo.Session, i *discordgo.InteractionCreate, content string, components []discordgo.MessageComponent) bool {
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content:    content,
			Flags:      discordgo.MessageFlagsEphemeral,
			Components: components,
		},
	}); err != nil {
		log.Printf("[Error] Failed to send interaction response with components: %v", err)
		return false
	}
	return true
}

func (b *Bot) deleteCategoryPicker(s *discordgo.Session, userID, channelID string) {
	value, ok := b.categoryPickerTokens.LoadAndDelete(categoryPickerKey(userID, channelID))
	if !ok {
		return
	}
	ref, ok := value.(interactionRef)
	if !ok {
		return
	}
	if err := s.InteractionResponseDelete(&discordgo.Interaction{
		AppID: ref.appID,
		Token: ref.token,
	}); err != nil {
		log.Printf("[Warning] Failed to delete ephemeral category picker for user %s: %v", userID, err)
	}
}

func (b *Bot) createTicketChannel(s *discordgo.Session, userID, username, category string) (*discordgo.Channel, error) {
	permissionOverwrites := []*discordgo.PermissionOverwrite{
		{
			ID:    b.cfg.GuildID,
			Type:  discordgo.PermissionOverwriteTypeRole,
			Deny:  discordgo.PermissionViewChannel,
			Allow: 0,
		},
		{
			ID:   userID,
			Type: discordgo.PermissionOverwriteTypeMember,
			Allow: discordgo.PermissionViewChannel |
				discordgo.PermissionSendMessages |
				discordgo.PermissionReadMessageHistory |
				discordgo.PermissionAttachFiles |
				discordgo.PermissionEmbedLinks,
			Deny: 0,
		},
	}

	if s.State != nil && s.State.User != nil && s.State.User.ID != "" {
		permissionOverwrites = append(permissionOverwrites, &discordgo.PermissionOverwrite{
			ID:   s.State.User.ID,
			Type: discordgo.PermissionOverwriteTypeMember,
			Allow: discordgo.PermissionViewChannel |
				discordgo.PermissionSendMessages |
				discordgo.PermissionReadMessageHistory |
				discordgo.PermissionAttachFiles |
				discordgo.PermissionEmbedLinks |
				discordgo.PermissionManageChannels |
				discordgo.PermissionManageMessages,
			Deny: 0,
		})
	}

	for _, roleID := range b.cfg.StaffRoleIDs {
		if strings.TrimSpace(roleID) != "" {
			permissionOverwrites = append(permissionOverwrites, &discordgo.PermissionOverwrite{
				ID:   roleID,
				Type: discordgo.PermissionOverwriteTypeRole,
				Allow: discordgo.PermissionViewChannel |
					discordgo.PermissionSendMessages |
					discordgo.PermissionReadMessageHistory |
					discordgo.PermissionAttachFiles |
					discordgo.PermissionEmbedLinks,
				Deny: 0,
			})
		}
	}

	channelName := fmt.Sprintf("ticket-%s", models.CategoryAbbreviation(category))

	channel, err := s.GuildChannelCreateComplex(b.cfg.GuildID, discordgo.GuildChannelCreateData{
		Name:                 channelName,
		Type:                 discordgo.ChannelTypeGuildText,
		ParentID:             b.cfg.CategoryID,
		PermissionOverwrites: permissionOverwrites,
		Topic:                fmt.Sprintf("Support ticket for %s (%s)", username, userID),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create ticket channel: %w", err)
	}
	return channel, nil
}

// handleCategoryButton opens the category modal from either the public panel or an old ticket.
func (b *Bot) handleCategoryButton(s *discordgo.Session, i *discordgo.InteractionCreate, customID string) {
	userID, _ := interactionUser(i)
	if userID == "" {
		respondEphemeral(s, i, "⚠️ Could not identify your Discord account. Please try again.")
		return
	}

	ticket, err := b.db.GetTicketByChannelID(i.ChannelID)
	if errors.Is(err, database.ErrTicketNotFound) {
		activeTickets, activeErr := b.db.CountActiveTicketsForUser(userID)
		if activeErr != nil {
			log.Printf("[Error] Failed to check active ticket for user %s: %v", userID, activeErr)
			respondEphemeral(s, i, "❌ Could not check your existing tickets. Please try again.")
			return
		}
		if activeTickets >= maxActiveTicketsPerUser {
			respondEphemeral(s, i, fmt.Sprintf("⚠️ You already have %d active tickets. The limit is %d; close one before creating another.", activeTickets, maxActiveTicketsPerUser))
			return
		}

		b.respondWithCategoryModal(s, i, customID)
		return
	}
	if err != nil {
		log.Printf("[Error] Failed to fetch ticket for channel %s: %v", i.ChannelID, err)
		respondEphemeral(s, i, "⚠️ Ticket not found for this channel.")
		return
	}

	if userID != ticket.UserID {
		respondEphemeral(s, i, "⚠️ Only the ticket creator can select the category.")
		return
	}

	if ticket.Status == models.StatusResolved || ticket.Status == models.StatusClosed {
		respondEphemeral(s, i, "⚠️ This ticket has already been submitted.")
		return
	}

	b.respondWithCategoryModal(s, i, customID)
}

func (b *Bot) respondWithCategoryModal(s *discordgo.Session, i *discordgo.InteractionCreate, customID string) {
	var modal *discordgo.InteractionResponse
	switch customID {
	case IDCategoryGriefing:
		modal = ModalGriefing()
	case IDCategoryShip:
		modal = ModalShip()
	case IDCategoryBug:
		modal = ModalBug()
	case IDCategoryCrash:
		modal = ModalCrash()
	case IDCategoryAppeal:
		modal = ModalAppeal()
	case IDCategoryOther:
		modal = ModalOther()
	default:
		return
	}

	if err := s.InteractionRespond(i.Interaction, modal); err != nil {
		log.Printf("[Error] Failed to open modal for channel %s: %v", i.ChannelID, err)
	}
}

type ticketModalValues struct {
	category        string
	nickname        string
	categoryDetails string
	additionalInfo  string
	problemDesc     string
}

func parseTicketModal(data discordgo.ModalSubmitInteractionData) (ticketModalValues, bool) {
	fields := make(map[string]string)
	for _, comp := range data.Components {
		switch outer := comp.(type) {
		case *discordgo.Label:
			if input, ok := outer.Component.(*discordgo.TextInput); ok {
				fields[input.CustomID] = strings.TrimSpace(input.Value)
			}
		case *discordgo.ActionsRow:
			for _, inner := range outer.Components {
				if input, ok := inner.(*discordgo.TextInput); ok {
					fields[input.CustomID] = strings.TrimSpace(input.Value)
				}
			}
		}
	}

	values := ticketModalValues{
		nickname:       fields[IDFieldNickname],
		additionalInfo: fields[IDFieldAdditionalInfo],
		problemDesc:    fields[IDFieldDescription],
	}

	switch data.CustomID {
	case IDModalGriefing:
		values.category = string(models.CategoryGriefing)
		values.categoryDetails = sanitizeCoordinates(fields[IDFieldCoords])
	case IDModalShip:
		values.category = string(models.CategoryShip)
		values.categoryDetails = fields[IDFieldShipDetails]
	case IDModalBug:
		values.category = string(models.CategoryBug)
	case IDModalCrash:
		values.category = string(models.CategoryCrash)
		values.categoryDetails = fields[IDFieldCrashLogs]
	case IDModalAppeal:
		values.category = string(models.CategoryAppeal)
		values.categoryDetails = fields[IDFieldPunishment]
		values.problemDesc = fields[IDFieldAppealReason]
	case IDModalOther:
		values.category = string(models.CategoryOther)
	default:
		return values, false
	}

	return values, true
}

func (v ticketModalValues) validationError() string {
	if v.nickname == "" {
		return "⚠️ Please enter your in-game nickname."
	}

	switch v.category {
	case string(models.CategoryGriefing):
		if v.categoryDetails == "" {
			return "⚠️ Please enter the incident coordinates."
		}
	case string(models.CategoryShip):
		if v.categoryDetails == "" {
			return "⚠️ Please enter the ship claim name."
		}
	case string(models.CategoryCrash):
		if v.categoryDetails == "" {
			return "⚠️ Please paste the crash logs."
		}
	case string(models.CategoryAppeal):
		if v.categoryDetails == "" {
			return "⚠️ Please tell us which punishment you are appealing."
		}
		if v.problemDesc == "" {
			return "⚠️ Please explain why your punishment should be revoked."
		}
	case string(models.CategoryBug), string(models.CategoryOther):
		if v.problemDesc == "" {
			return "⚠️ Please describe the problem."
		}
	}

	return ""
}

// handleModalSubmit processes the submitted modal form for each ticket category.
func (b *Bot) handleModalSubmit(s *discordgo.Session, i *discordgo.InteractionCreate) {
	values, ok := parseTicketModal(i.ModalSubmitData())
	if !ok {
		log.Printf("[Warning] Unknown modal custom ID: %s", i.ModalSubmitData().CustomID)
		respondEphemeral(s, i, "⚠️ Unknown form submitted. Please try again.")
		return
	}

	existing, err := b.db.GetTicketByChannelID(i.ChannelID)
	if err != nil && !errors.Is(err, database.ErrTicketNotFound) {
		log.Printf("[Error] Failed to fetch ticket for channel %s: %v", i.ChannelID, err)
		respondEphemeral(s, i, "⚠️ Failed to load your ticket. Please try again.")
		return
	}

	if existing != nil {
		b.handleExistingTicketModal(s, i, existing, values)
		return
	}

	b.handleNewTicketModal(s, i, values)
}

func (b *Bot) handleExistingTicketModal(s *discordgo.Session, i *discordgo.InteractionCreate, ticket *models.Ticket, values ticketModalValues) {
	userID, _ := interactionUser(i)
	if userID != ticket.UserID {
		respondEphemeral(s, i, "⚠️ Only the ticket creator can submit this form.")
		return
	}
	if ticket.Status == models.StatusResolved || ticket.Status == models.StatusClosed {
		respondEphemeral(s, i, "⚠️ This ticket has already been submitted.")
		return
	}
	if message := values.validationError(); message != "" {
		respondEphemeral(s, i, message)
		return
	}

	if err := b.db.UpdateTicketInfo(i.ChannelID, values.category, values.nickname, values.categoryDetails, values.additionalInfo, values.problemDesc); err != nil {
		log.Printf("[Error] Failed to update ticket info for channel %s: %v", i.ChannelID, err)
		respondEphemeral(s, i, "⚠️ Failed to save your ticket details. Please try again.")
		return
	}

	updated, err := b.db.GetTicketByChannelID(i.ChannelID)
	if err != nil {
		log.Printf("[Error] Failed to fetch ticket after update for channel %s: %v", i.ChannelID, err)
		updated = &models.Ticket{
			ChannelID:          i.ChannelID,
			UserID:             userID,
			Category:           values.category,
			IngameNickname:     values.nickname,
			CategoryDetails:    values.categoryDetails,
			AdditionalInfo:     values.additionalInfo,
			ProblemDescription: values.problemDesc,
			Status:             models.StatusResolved,
		}
	}

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{TicketSummaryEmbed(updated, b.cfg.StaffRoleIDs)},
			Components: TicketActionComponents(updated),
		},
	}); err != nil {
		log.Printf("[Error] Failed to send ticket summary for channel %s: %v", i.ChannelID, err)
	}

	b.disableCategoryButtons(s, i.ChannelID)
	b.deleteCategoryPicker(s, userID, i.ChannelID)
}

func (b *Bot) handleNewTicketModal(s *discordgo.Session, i *discordgo.InteractionCreate, values ticketModalValues) {
	userID, username := interactionUser(i)
	if userID == "" {
		respondEphemeral(s, i, "⚠️ Could not identify your Discord account. Please try again.")
		return
	}
	if message := values.validationError(); message != "" {
		respondEphemeral(s, i, message)
		return
	}

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	}); err != nil {
		log.Printf("[Error] Failed to defer modal response for user %s: %v", userID, err)
		return
	}

	respond := func(content string) {
		if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &content}); err != nil {
			log.Printf("[Error] Failed to edit modal response for user %s: %v", userID, err)
		}
	}

	// Serialize creation per user so concurrent submissions cannot bypass the active-ticket limit.
	creationLock := b.ticketCreationLock(userID)
	creationLock.Lock()
	defer creationLock.Unlock()

	if remaining, ok := b.reserveTicketCreation(userID); !ok {
		seconds := int(remaining.Seconds()) + 1
		respond(fmt.Sprintf("⏳ Please wait %d seconds before creating another ticket.", seconds))
		return
	}

	ticketCreated := false
	defer func() {
		if !ticketCreated {
			b.releaseTicketCreation(userID)
		}
	}()

	activeTickets, err := b.db.CountActiveTicketsForUser(userID)
	if err != nil {
		log.Printf("[Error] Failed to check active ticket for user %s: %v", userID, err)
		respond("❌ Could not check your existing tickets. Please try again.")
		return
	}
	if activeTickets >= maxActiveTicketsPerUser {
		respond(fmt.Sprintf("⚠️ You already have %d active tickets. The limit is %d; close one before creating another.", activeTickets, maxActiveTicketsPerUser))
		return
	}

	channel, err := b.createTicketChannel(s, userID, username, values.category)
	if err != nil {
		log.Printf("[Error] Failed to create ticket channel for user %s: %v", userID, err)
		respond("❌ Failed to create ticket channel. Please contact an administrator.")
		return
	}

	ticket, err := b.db.CreateTicketWithInfo(channel.ID, userID, values.category, values.nickname, values.categoryDetails, values.additionalInfo, values.problemDesc)
	if err != nil {
		log.Printf("[Error] Failed to create database ticket for channel %s: %v", channel.ID, err)
		if _, deleteErr := s.ChannelDelete(channel.ID); deleteErr != nil {
			log.Printf("[Warning] Failed to delete orphaned ticket channel %s: %v", channel.ID, deleteErr)
		}
		respond("❌ Database error when creating your ticket. Please try again.")
		return
	}
	ticketCreated = true

	channelName := models.TicketChannelName(ticket.ID, ticket.Category)
	if _, err := s.ChannelEdit(channel.ID, &discordgo.ChannelEdit{Name: channelName}); err != nil {
		log.Printf("[Warning] Failed to rename ticket channel %s to %s: %v", channel.ID, channelName, err)
	}

	content := ""
	if mention := b.availableStaffRoleMention(); mention != "" {
		content = fmt.Sprintf("🔔 %s — a new ticket needs attention.\n\n", mention)
	}
	content += fmt.Sprintf("👋 Welcome <@%s>! **Ticket #%d** has been opened.\n\n", userID, ticket.ID) +
		"Please do not ping staff for personal matters or routine ticket inquiries.\n\n" +
		"Direct pings are strictly reserved for critical emergencies that impact the entire server (such as major downtime, severe network issues, or ongoing malicious attacks).\n\n" +
		"Pinging staff for non-urgent requests will not result in a faster response and may delay your support. A team member will assist you as soon as possible. Thank you for your patience and cooperation!"

	if _, err := s.ChannelMessageSendComplex(channel.ID, &discordgo.MessageSend{
		Content:    content,
		Embeds:     []*discordgo.MessageEmbed{TicketSummaryEmbed(ticket, b.cfg.StaffRoleIDs)},
		Components: TicketActionComponents(ticket),
	}); err != nil {
		log.Printf("[Error] Failed to send ticket summary in channel %s: %v", channel.ID, err)
	}

	b.deleteCategoryPicker(s, userID, i.ChannelID)

	if err := s.InteractionResponseDelete(i.Interaction); err != nil {
		log.Printf("[Error] Failed to delete deferred modal response for user %s: %v", userID, err)
	}
}

// disableCategoryButtons edits the category prompt message to remove its buttons after submission
func (b *Bot) disableCategoryButtons(s *discordgo.Session, channelID string) {
	// Preferred path: we remembered the exact prompt message ID when it was sent.
	if stored, ok := b.categoryPrompt.Load(channelID); ok {
		if msgID, _ := stored.(string); msgID != "" {
			_, err := s.ChannelMessageEditComplex(&discordgo.MessageEdit{
				Channel:    channelID,
				ID:         msgID,
				Components: &[]discordgo.MessageComponent{},
			})
			if err == nil {
				b.categoryPrompt.Delete(channelID)
				return
			}
			log.Printf("[Warning] Failed to disable category buttons on message %s: %v", msgID, err)
		}
	}

	// Fallback: scan recent history for the prompt message.
	msgs, err := s.ChannelMessages(channelID, 100, "", "", "")
	if err != nil {
		return
	}
	botID := s.State.User.ID
	for _, msg := range msgs {
		if msg.Author.ID == botID && len(msg.Components) > 0 {
			for _, row := range msg.Components {
				actionRow, ok := row.(*discordgo.ActionsRow)
				if !ok {
					continue
				}
				for _, comp := range actionRow.Components {
					btn, ok := comp.(*discordgo.Button)
					if !ok {
						continue
					}
					if btn.CustomID == IDCategoryGriefing || btn.CustomID == IDCategoryShip || btn.CustomID == IDCategoryAppeal {
						s.ChannelMessageEditComplex(&discordgo.MessageEdit{
							Channel:    channelID,
							ID:         msg.ID,
							Components: &[]discordgo.MessageComponent{},
						})
						b.categoryPrompt.Delete(channelID)
						return
					}
				}
			}
		}
	}
}

// authorizeTicketClosure ensures only the ticket creator or staff can close a ticket.
func (b *Bot) authorizeTicketClosure(s *discordgo.Session, i *discordgo.InteractionCreate) (*models.Ticket, bool) {
	ticket, err := b.db.GetTicketByChannelID(i.ChannelID)
	if err != nil {
		if errors.Is(err, database.ErrTicketNotFound) {
			respondEphemeral(s, i, "⚠️ This channel is not an active support ticket channel.")
		} else {
			log.Printf("[Error] Failed to load ticket for channel %s: %v", i.ChannelID, err)
			respondEphemeral(s, i, "❌ Could not load this ticket. Please try again.")
		}
		return nil, false
	}

	userID, _ := interactionUser(i)
	if userID != ticket.UserID && !b.isStaffInteraction(i) {
		respondEphemeral(s, i, "⚠️ Only the ticket creator or staff can close this ticket.")
		return nil, false
	}

	return ticket, true
}

// handleCloseRequestButton prompts with confirmation buttons
func (b *Bot) handleCloseRequestButton(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if _, ok := b.authorizeTicketClosure(s, i); !ok {
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content:    "⚠️ **Are you sure you want to close this ticket?**\nA transcript will be generated and archived before this channel is deleted.",
			Components: CloseConfirmationComponents(),
		},
	})
}

// handleCloseCancelButton dismisses the closure prompt
func (b *Bot) handleCloseCancelButton(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if _, ok := b.authorizeTicketClosure(s, i); !ok {
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    "❌ Ticket closure cancelled.",
			Components: []discordgo.MessageComponent{},
		},
	})
}

// handleCloseConfirmButton initiates transcript generation and channel deletion
func (b *Bot) handleCloseConfirmButton(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if _, ok := b.authorizeTicketClosure(s, i); !ok {
		return
	}

	// Guard against a double-click generating duplicate transcripts/uploads.
	if _, loaded := b.closing.LoadOrStore(i.ChannelID, struct{}{}); loaded {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "⏳ This ticket is already being closed.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    "⏳ **Closing ticket and generating transcript...**",
			Components: []discordgo.MessageComponent{},
		},
	})

	userTag := ""
	if i.Member != nil && i.Member.User != nil {
		userTag = i.Member.User.Username
		if i.Member.User.Discriminator != "0" && i.Member.User.Discriminator != "" {
			userTag = fmt.Sprintf("%s#%s", i.Member.User.Username, i.Member.User.Discriminator)
		}
	}

	go b.closeTicketAndGenerateTranscript(s, i.ChannelID, userTag)
}
