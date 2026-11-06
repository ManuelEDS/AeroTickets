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

	if ticket.Status == models.StatusClosed {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "⚠️ This ticket is already closed.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
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
