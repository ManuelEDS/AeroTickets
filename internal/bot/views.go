package bot

import (
	"fmt"
	"time"

	"aerobot/internal/models"

	"github.com/bwmarrin/discordgo"
)

// UI Component / Modal Custom IDs
const (
	// Buttons
	IDTicketCreate       = "ticket_create"
	IDCategoryGriefing   = "ticket_cat_griefing"
	IDCategoryShip       = "ticket_cat_ship"
	IDCategoryBug        = "ticket_cat_bug"
	IDCategoryCrash      = "ticket_cat_crash"
	IDCategoryAppeal     = "ticket_cat_appeal"
	IDCategoryOther      = "ticket_cat_other"
	IDTicketCloseRequest = "ticket_close_request"
	IDTicketCloseConfirm = "ticket_close_confirm"
	IDTicketCloseCancel  = "ticket_close_cancel"
	IDTicketTeleportCopy = "ticket_teleport_copy"

	// Modals
	IDModalGriefing = "modal_griefing"
	IDModalShip     = "modal_ship"
	IDModalBug      = "modal_bug"
	IDModalCrash    = "modal_crash"
	IDModalAppeal   = "modal_appeal"
	IDModalOther    = "modal_other"

	// Modal field IDs
	IDFieldNickname       = "field_nickname"
	IDFieldCoords         = "field_coords"
	IDFieldShipDetails    = "field_ship_details"
	IDFieldCrashLogs      = "field_crash_logs"
	IDFieldPunishment     = "field_punishment"
	IDFieldAppealReason   = "field_appeal_reason"
	IDFieldAdditionalInfo = "field_additional_info"
	IDFieldDescription    = "field_description"
)

func boolPtr(value bool) *bool {
	return &value
}

// Brand colors
const (
	ColorPrimary = 0x5865F2 // Blurple
	ColorSuccess = 0x57F287 // Green
	ColorWarning = 0xFEE75C // Yellow
	ColorDanger  = 0xED4245 // Red
	ColorDark    = 0x2B2D31 // Charcoal
)

// TicketPanelEmbed returns the main public panel embed
func TicketPanelEmbed() *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       "🎫 Create a Support Ticket",
		Description: "Select the category below that best matches your issue.\nYou will fill out a short form, and your private ticket channel will be created automatically after you submit it.",
		Color:       ColorPrimary,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "📌 Categories",
				Value:  "• Griefing / Stealing\n• Ship Related\n• Bug / Exploit Report\n• Crash Report\n• Appeals\n• General Support / Other",
				Inline: false,
			},
			{
				Name:   "📝 How it works",
				Value:  "1. Press the category button for your issue.\n2. Fill in the short form.\n3. Your private ticket channel is created automatically after submission.",
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "AeroBot Ticket System",
		},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
}

// TicketPanelComponents returns the category buttons shown in the public panel.
func TicketPanelComponents() []discordgo.MessageComponent {
	return CategorySelectionComponents()
}

// CategorySelectionComponents returns the 6 category buttons in action rows
func CategorySelectionComponents() []discordgo.MessageComponent {
	return []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "Griefing / Stealing",
					Style:    discordgo.PrimaryButton,
					CustomID: IDCategoryGriefing,
					Emoji: &discordgo.ComponentEmoji{
						Name: "⚔️",
					},
				},
				discordgo.Button{
					Label:    "Ship Related",
					Style:    discordgo.PrimaryButton,
					CustomID: IDCategoryShip,
					Emoji: &discordgo.ComponentEmoji{
						Name: "🚀",
					},
				},
				discordgo.Button{
					Label:    "Bug / Exploit Report",
					Style:    discordgo.PrimaryButton,
					CustomID: IDCategoryBug,
					Emoji: &discordgo.ComponentEmoji{
						Name: "🐛",
					},
				},
			},
		},
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "Crash Report",
					Style:    discordgo.PrimaryButton,
					CustomID: IDCategoryCrash,
					Emoji: &discordgo.ComponentEmoji{
						Name: "💥",
					},
				},
				discordgo.Button{
					Label:    "Appeals",
					Style:    discordgo.PrimaryButton,
					CustomID: IDCategoryAppeal,
					Emoji: &discordgo.ComponentEmoji{
						Name: "⚖️",
					},
				},
				discordgo.Button{
					Label:    "Other",
					Style:    discordgo.SecondaryButton,
					CustomID: IDCategoryOther,
					Emoji: &discordgo.ComponentEmoji{
						Name: "❓",
					},
				},
			},
		},
	}
}

// CloseTicketButtonComponent returns the "Close Ticket" button
func CloseTicketButtonComponent() []discordgo.MessageComponent {
	return []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "Close Ticket",
					Style:    discordgo.DangerButton,
					CustomID: IDTicketCloseRequest,
					Emoji: &discordgo.ComponentEmoji{
						Name: "🔴",
					},
				},
			},
		},
	}
}

// TicketActionComponents returns the controls available on a ticket summary.
func TicketActionComponents(ticket *models.Ticket) []discordgo.MessageComponent {
	buttons := []discordgo.MessageComponent{
		discordgo.Button{
			Label:    "Close Ticket",
			Style:    discordgo.DangerButton,
			CustomID: IDTicketCloseRequest,
			Emoji:    &discordgo.ComponentEmoji{Name: "🔴"},
		},
	}
	if ticket.Category == string(models.CategoryGriefing) && ticket.CategoryDetails != "" {
		buttons = append(buttons, discordgo.Button{
			Label:    "Copy Teleport Command",
			Style:    discordgo.PrimaryButton,
			CustomID: IDTicketTeleportCopy,
			Emoji:    &discordgo.ComponentEmoji{Name: "📋"},
		})
	}
	return []discordgo.MessageComponent{discordgo.ActionsRow{Components: buttons}}
}

// CloseConfirmationComponents returns the confirm / cancel buttons
func CloseConfirmationComponents() []discordgo.MessageComponent {
	return []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "Confirm Closure",
					Style:    discordgo.DangerButton,
					CustomID: IDTicketCloseConfirm,
					Emoji: &discordgo.ComponentEmoji{
						Name: "⚠️",
					},
				},
				discordgo.Button{
					Label:    "Cancel",
					Style:    discordgo.SecondaryButton,
					CustomID: IDTicketCloseCancel,
					Emoji: &discordgo.ComponentEmoji{
						Name: "❌",
					},
				},
			},
		},
	}
}

func textInputField(label, description, customID, placeholder string, style discordgo.TextInputStyle, required bool, maxLength int) discordgo.Label {
	return discordgo.Label{
		Label:       label,
		Description: description,
		Component: discordgo.TextInput{
			CustomID:    customID,
			Style:       style,
			Placeholder: placeholder,
			Required:    boolPtr(required),
			MaxLength:   maxLength,
		},
	}
}

func ModalGriefing() *discordgo.InteractionResponse {
	return &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: IDModalGriefing,
			Title:    "⚔️ Griefing / Stealing Report",
			Flags:    discordgo.MessageFlagsIsComponentsV2,
			Components: []discordgo.MessageComponent{
				textInputField("In-Game Nickname", "Enter the nickname you use in-game.", IDFieldNickname, "e.g. StarPilot99", discordgo.TextInputShort, true, 64),
				textInputField("Incident Coordinates", "Coordinates where the incident happened.", IDFieldCoords, "e.g. X: 1024, Y: -300, Z: 512", discordgo.TextInputShort, true, 128),
				textInputField("Additional Info (Optional)", "Add any other useful context.", IDFieldAdditionalInfo, "Anything else staff should know...", discordgo.TextInputParagraph, false, 1000),
			},
		},
	}
}

func ModalShip() *discordgo.InteractionResponse {
	return &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: IDModalShip,
			Title:    "🚀 Ship Related Issue",
			Flags:    discordgo.MessageFlagsIsComponentsV2,
			Components: []discordgo.MessageComponent{
				textInputField("In-Game Nickname", "Enter the nickname you use in-game.", IDFieldNickname, "e.g. StarPilot99", discordgo.TextInputShort, true, 64),
				textInputField("Ship Claim Name", "Enter the claim name of the ship.", IDFieldShipDetails, "e.g. Aurora MR", discordgo.TextInputShort, true, 128),
				textInputField("Additional Info (Optional)", "Add any other useful context.", IDFieldAdditionalInfo, "Anything else staff should know...", discordgo.TextInputParagraph, false, 1000),
			},
		},
	}
}

func ModalBug() *discordgo.InteractionResponse {
	return &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: IDModalBug,
			Title:    "🐛 Bug / Exploit Report",
			Flags:    discordgo.MessageFlagsIsComponentsV2,
			Components: []discordgo.MessageComponent{
				textInputField("In-Game Nickname", "Enter the nickname you use in-game.", IDFieldNickname, "e.g. StarPilot99", discordgo.TextInputShort, true, 64),
				textInputField("Problem Description", "Describe the bug or exploit in detail.", IDFieldDescription, "Steps to reproduce, expected vs. actual behavior...", discordgo.TextInputParagraph, true, 1000),
			},
		},
	}
}

func ModalCrash() *discordgo.InteractionResponse {
	return &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: IDModalCrash,
			Title:    "💥 Crash Report",
			Flags:    discordgo.MessageFlagsIsComponentsV2,
			Components: []discordgo.MessageComponent{
				textInputField("In-Game Nickname", "Enter the nickname you use in-game.", IDFieldNickname, "e.g. StarPilot99", discordgo.TextInputShort, true, 64),
				textInputField("Crash Logs", "Paste the crash log output or a link to it.", IDFieldCrashLogs, "Paste crash logs here...", discordgo.TextInputParagraph, true, 4000),
				textInputField("Additional Info (Optional)", "Add any other useful context.", IDFieldAdditionalInfo, "Anything else staff should know...", discordgo.TextInputParagraph, false, 1000),
			},
		},
	}
}

func ModalAppeal() *discordgo.InteractionResponse {
	return &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: IDModalAppeal,
			Title:    "⚖️ Punishment Appeal",
			Flags:    discordgo.MessageFlagsIsComponentsV2,
			Components: []discordgo.MessageComponent{
				textInputField("In-Game Nickname", "Enter the nickname you use in-game.", IDFieldNickname, "e.g. StarPilot99", discordgo.TextInputShort, true, 64),
				textInputField("What punishment are you appealing?", "Include the punishment reason stated in the punishment message.", IDFieldPunishment, "e.g. griefing / stealing / etc", discordgo.TextInputShort, true, 128),
				textInputField("Why should it be revoked?", "Explain why you think your punishment needs to be revoked. Include any evidence or context.", IDFieldAppealReason, "Tell us what happened and why the punishment should be lifted...", discordgo.TextInputParagraph, true, 1000),
			},
		},
	}
}

func ModalOther() *discordgo.InteractionResponse {
	return &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: IDModalOther,
			Title:    "❓ General Support / Other",
			Flags:    discordgo.MessageFlagsIsComponentsV2,
			Components: []discordgo.MessageComponent{
				textInputField("In-Game Nickname", "Enter the nickname you use in-game.", IDFieldNickname, "e.g. StarPilot99", discordgo.TextInputShort, true, 64),
				textInputField("Problem Description", "Describe what you need help with.", IDFieldDescription, "Tell us what you need help with...", discordgo.TextInputParagraph, true, 1000),
			},
		},
	}
}

// TicketSummaryEmbed returns an embed summarizing the collected ticket information
func TicketSummaryEmbed(ticket *models.Ticket, staffRoleIDs []string) *discordgo.MessageEmbed {
	fields := []*discordgo.MessageEmbedField{
		{
			Name:   "👤 User",
			Value:  fmt.Sprintf("<@%s>", ticket.UserID),
			Inline: true,
		},
		{
			Name:   "🎮 In-Game Nickname",
			Value:  ticket.IngameNickname,
			Inline: true,
		},
		{
			Name:   "📁 Category",
			Value:  ticket.Category,
			Inline: true,
		},
	}

	categoryDetails := ticket.CategoryDetails
	additionalInfo := ticket.AdditionalInfo
	if categoryDetails == "" && (ticket.Category == string(models.CategoryGriefing) || ticket.Category == string(models.CategoryShip)) {
		categoryDetails = additionalInfo
		additionalInfo = ""
	}

	if categoryDetails != "" {
		label := models.CategoryDetailLabel(ticket.Category)
		if label == "" {
			label = "Details"
		}
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   label,
			Value:  categoryDetails,
			Inline: false,
		})
	}

	if additionalInfo != "" {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   "ℹ️ Additional Info",
			Value:  additionalInfo,
			Inline: false,
		})
	}

	if ticket.ProblemDescription != "" {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   "📝 " + models.CategoryProblemLabel(ticket.Category),
			Value:  ticket.ProblemDescription,
			Inline: false,
		})
	}

	return &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("🎫 Ticket #%d Details", ticket.ID),
		Description: "Your details have been registered! A team member will assist you as soon as possible.",
		Color:       ColorSuccess,
		Fields:      fields,
		Footer: &discordgo.MessageEmbedFooter{
			Text: "AeroBot Ticket System",
		},
		Timestamp: ticket.CreatedAt.Format(time.RFC3339),
	}
}

// TranscriptLogEmbed returns an embed to post in the transcript channel alongside the .md file
func TranscriptLogEmbed(ticket *models.Ticket, closedBy string, channelName string) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("📄 Transcript: #%s", channelName),
		Description: fmt.Sprintf("Ticket **#%d** has been closed.", ticket.ID),
		Color:       ColorPrimary,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Ticket ID",
				Value:  fmt.Sprintf("%d", ticket.ID),
				Inline: true,
			},
			{
				Name:   "Owner",
				Value:  fmt.Sprintf("<@%s>", ticket.UserID),
				Inline: true,
			},
			{
				Name:   "Closed By",
				Value:  closedBy,
				Inline: true,
			},
			{
				Name:   "In-Game Nickname",
				Value:  ticket.IngameNickname,
				Inline: true,
			},
			{
				Name:   "Category",
				Value:  ticket.Category,
				Inline: true,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "AeroBot Transcript Archive",
		},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
}

// UserTranscriptClosedEmbed returns an embed sent to the ticket creator via DM upon closure
func UserTranscriptClosedEmbed(ticket *models.Ticket, channelName string, fileName string) *discordgo.MessageEmbed {
	category := ticket.Category
	if category == "" {
		category = "N/A"
	}
	nickname := ticket.IngameNickname
	if nickname == "" {
		nickname = "N/A"
	}

	return &discordgo.MessageEmbed{
		Title: "🎫 Your Support Ticket Has Been Closed",
		Description: fmt.Sprintf(
			"Hello! Your support ticket **#%d** (`#%s`) has been marked as **closed**.\n\n"+
				"📄 **Transcript File Attached:** The attached `.md` (**%s**) file is the complete transcript of your support ticket, "+
				"including the full message history, questions, answers, and staff responses for your personal records.",
			ticket.ID, channelName, fileName,
		),
		Color: ColorPrimary,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Ticket ID",
				Value:  fmt.Sprintf("%d", ticket.ID),
				Inline: true,
			},
			{
				Name:   "Category",
				Value:  category,
				Inline: true,
			},
			{
				Name:   "In-Game Nickname",
				Value:  nickname,
				Inline: true,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "AeroBot Ticket System • Thank you for contacting support",
		},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
}
