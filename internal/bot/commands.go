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
	}
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
