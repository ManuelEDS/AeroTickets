package bot

import (
	"fmt"
	"log"

	"aerobot/internal/models"
	"github.com/bwmarrin/discordgo"
)

var defaultAdminPerms int64 = discordgo.PermissionManageChannels | discordgo.PermissionAdministrator

var ApplicationCommands = []*discordgo.ApplicationCommand{
	{
		Name:                     "setup-panel",
		Description:              "Deploy the interactive Ticket Creation Panel into this channel",
		DefaultMemberPermissions: &defaultAdminPerms,
	},
	{
		Name:        "close",
		Description: "Request to close the current support ticket",
	},
	{
		Name:        "ticket-info",
		Description: "View information about the current support ticket",
	},
	{
		Name:        "add-user",
		Description: "Give a user access to the current support ticket",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "user",
				Description: "The user to add to this ticket",
				Type:        discordgo.ApplicationCommandOptionUser,
				Required:    true,
			},
		},
	},
	{
		Name:        "available",
		Description: "Get the \"available staff\" role and receive pings for new tickets",
	},
	{
		Name:        "unavailable",
		Description: "Remove the \"available staff\" role and stop receiving new ticket pings",
	},
}

// handleSlashCommand handles slash command executions
func (b *Bot) handleSlashCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	cmdData := i.ApplicationCommandData()

	switch cmdData.Name {
	case "setup-panel":
		b.cmdSetupPanel(s, i)
	case "close":
		b.cmdCloseTicket(s, i)
	case "ticket-info":
		b.cmdTicketInfo(s, i)
	case "add-user":
		b.cmdAddTicketUser(s, i, cmdData.Options)
	case "available":
		b.cmdSetAvailability(s, i, true)
	case "unavailable":
		b.cmdSetAvailability(s, i, false)
	}
}

// isEligibleForAvailability reports whether the member holds one of the configured staff roles.
func (b *Bot) isEligibleForAvailability(i *discordgo.InteractionCreate) bool {
	if i.Member == nil {
		return false
	}
	for _, memberRoleID := range i.Member.Roles {
		for _, staffRoleID := range b.cfg.StaffRoleIDs {
			if memberRoleID == staffRoleID {
				return true
			}
		}
	}
	return false
}

// cmdSetAvailability lets eligible staff add or remove the "available staff" role.
func (b *Bot) cmdSetAvailability(s *discordgo.Session, i *discordgo.InteractionCreate, available bool) {
	if !b.isEligibleForAvailability(i) {
		respondEphemeral(s, i, "⚠️ Only members with a configured staff role can manage the \"available staff\" role.")
		return
	}

	roleID := b.getAvailableStaffRoleID()
	if roleID == "" {
		respondEphemeral(s, i, "❌ The \"available staff\" role is not ready yet. Please contact an administrator.")
		return
	}

	userID := i.Member.User.ID
	hasRole := false
	for _, memberRoleID := range i.Member.Roles {
		if memberRoleID == roleID {
			hasRole = true
			break
		}
	}

	if available == hasRole {
		if available {
			respondEphemeral(s, i, "ℹ️ You already have the \"available staff\" role.")
		} else {
			respondEphemeral(s, i, "ℹ️ You already do not have the \"available staff\" role.")
		}
		return
	}

	var err error
	if available {
		err = s.GuildMemberRoleAdd(i.GuildID, userID, roleID)
	} else {
		err = s.GuildMemberRoleRemove(i.GuildID, userID, roleID)
	}
	if err != nil {
		log.Printf("[Error] Failed to update availability role for user %s: %v", userID, err)
		respondEphemeral(s, i, "❌ Failed to update the \"available staff\" role. Make sure the bot can manage roles and its role is above \"available staff\".")
		return
	}

	if available {
		respondEphemeral(s, i, "✅ You are now marked as available. You will be pinged for new tickets.")
	} else {
		respondEphemeral(s, i, "✅ You are no longer marked as available. You will not be pinged for new tickets.")
	}
}

func (b *Bot) cmdAddTicketUser(s *discordgo.Session, i *discordgo.InteractionCreate, options []*discordgo.ApplicationCommandInteractionDataOption) {
	if !b.isStaffInteraction(i) {
		respondEphemeral(s, i, "⚠️ Only configured ticket staff or members with Manage Channels/Administrator can add users.")
		return
	}

	ticket, err := b.db.GetTicketByChannelID(i.ChannelID)
	if err != nil {
		respondEphemeral(s, i, "⚠️ This command can only be used in a support ticket channel.")
		return
	}
	if ticket.Status == models.StatusClosed {
		respondEphemeral(s, i, "⚠️ This ticket is already closed.")
		return
	}
	if len(options) == 0 || options[0] == nil {
		respondEphemeral(s, i, "⚠️ Select a user to add to this ticket.")
		return
	}
	user := options[0].UserValue(nil)
	if user == nil || user.ID == "" {
		respondEphemeral(s, i, "⚠️ Could not identify the selected user.")
		return
	}

	permissions := int64(discordgo.PermissionViewChannel) |
		discordgo.PermissionSendMessages |
		discordgo.PermissionReadMessageHistory |
		discordgo.PermissionAttachFiles |
		discordgo.PermissionEmbedLinks
	if err := s.ChannelPermissionSet(i.ChannelID, user.ID, discordgo.PermissionOverwriteTypeMember, permissions, 0); err != nil {
		log.Printf("[Error] Failed to add user %s to ticket channel %s: %v", user.ID, i.ChannelID, err)
		respondEphemeral(s, i, "❌ Failed to add that user. Check that the bot can manage channel permissions and try again.")
		return
	}

	respondEphemeral(s, i, fmt.Sprintf("✅ <@%s> can now access this ticket.", user.ID))
}

func (b *Bot) isStaffInteraction(i *discordgo.InteractionCreate) bool {
	if i.Member == nil {
		return false
	}
	if i.Member.Permissions&(discordgo.PermissionAdministrator|discordgo.PermissionManageChannels) != 0 {
		return true
	}
	for _, memberRoleID := range i.Member.Roles {
		for _, staffRoleID := range b.cfg.StaffRoleIDs {
			if memberRoleID == staffRoleID {
				return true
			}
		}
	}
	return false
}

func (b *Bot) cmdSetupPanel(s *discordgo.Session, i *discordgo.InteractionCreate) {
	msg := &discordgo.MessageSend{
		Embeds:     []*discordgo.MessageEmbed{TicketPanelEmbed()},
		Components: TicketPanelComponents(),
	}

	_, err := s.ChannelMessageSendComplex(i.ChannelID, msg)
	if err != nil {
		log.Printf("[Error] Failed to send panel in channel %s: %v", i.ChannelID, err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("❌ Failed to deploy panel: %v", err),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "✅ Ticket Creation Panel has been deployed to this channel!",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func (b *Bot) cmdCloseTicket(s *discordgo.Session, i *discordgo.InteractionCreate) {
	ticket, orphan, ok := b.authorizeTicketClosure(s, i)
	if !ok {
		return
	}

	if !orphan && ticket.Status == models.StatusClosed {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "⚠️ This ticket is already closed.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	content := "⚠️ **Are you sure you want to close this ticket?**\nA transcript will be generated and archived before this channel is deleted."
	if orphan {
		content = "⚠️ **Are you sure you want to close this ticket?**\nThis channel has no database record, so no transcript will be generated. The channel will be deleted."
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content:    content,
			Components: CloseConfirmationComponents(),
		},
	})
}

func (b *Bot) cmdTicketInfo(s *discordgo.Session, i *discordgo.InteractionCreate) {
	ticket, err := b.db.GetTicketByChannelID(i.ChannelID)
	if err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "⚠️ This channel is not an active support ticket channel.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{TicketSummaryEmbed(ticket, b.cfg.StaffRoleIDs)},
			Flags:  discordgo.MessageFlagsEphemeral,
		},
	})
}
