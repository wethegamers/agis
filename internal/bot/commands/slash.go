package commands

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// RegisterSlashCommands registers Discord slash commands for all existing text commands.
// If guildID is non-empty, commands are registered in that guild for faster propagation;
// otherwise they are registered globally (may take up to 1 hour to appear).
func (h *CommandHandler) RegisterSlashCommands(s *discordgo.Session, guildID string) ([]*discordgo.ApplicationCommand, error) {
	// Determine application ID safely
	appID := ""
	if s != nil && s.State != nil && s.State.User != nil && s.State.User.ID != "" {
		appID = s.State.User.ID
	} else if h != nil && h.config != nil && h.config.Discord.ClientID != "" {
		appID = h.config.Discord.ClientID
	}
	if appID == "" {
		log.Printf("Skipping slash command registration: no application ID available (session not open and DISCORD_CLIENT_ID not set)")
		return nil, nil
	}

	created := make([]*discordgo.ApplicationCommand, 0, len(h.commands))

	for name, cmd := range h.commands {
		// Build a description and ensure it meets Discord length constraints
		desc := cmd.Description()
		if desc == "" {
			desc = fmt.Sprintf("Execute %s", name)
		}
		if len(desc) > 100 {
			desc = desc[:100]
		}

		ac := &discordgo.ApplicationCommand{
			Name:        name,
			Description: desc,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "args",
					Description: "Arguments for the command (space-separated)",
					Required:    false,
				},
			},
		}

		var newCmd *discordgo.ApplicationCommand
		var err error
		if guildID != "" {
			newCmd, err = s.ApplicationCommandCreate(appID, guildID, ac)
		} else {
			newCmd, err = s.ApplicationCommandCreate(appID, "", ac)
		}
		if err != nil {
			log.Printf("Failed to register slash command %s: %v", name, err)
			continue
		}
		created = append(created, newCmd)
	}
	return created, nil
}

// HandleInteraction routes slash commands to the existing command implementations.
func (h *CommandHandler) HandleInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	data := i.ApplicationCommandData()
	name := strings.ToLower(data.Name)
	var args []string
	if len(data.Options) > 0 {
		// We use a single "args" option; split on spaces to feed existing handlers
		if data.Options[0].StringValue() != "" {
			args = strings.Fields(data.Options[0].StringValue())
		}
	}

	// Acknowledge immediately to avoid the 3s timeout
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})

	// Build a minimal MessageCreate to reuse existing command flow
	fakeMsg := &discordgo.MessageCreate{
		Message: &discordgo.Message{
			ID:        "slash-command",
			ChannelID: i.ChannelID,
			GuildID:   i.GuildID,
			Author:    i.Member.User,
			Content:   name + " " + strings.Join(args, " "),
		},
	}

	// Look up command and execute
	if cmd, ok := h.commands[name]; ok {
		// Determine user permission
		userPerm := h.permissions.GetUserPermission(s, i.GuildID, i.Member.User.ID)
		if userPerm < cmd.RequiredPermission() {
			// Edit original response with permission error
			_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: strPtr("❌ Permission denied for this command.")})
			return
		}

		ctx := &CommandContext{
			Session:        s,
			Message:        fakeMsg,
			Args:           args,
			DB:             h.db,
			Config:         h.config,
			Permissions:    h.permissions,
			UserPerm:       userPerm,
			Logger:         h.logger,
			Context:        context.Background(),
			EnhancedServer: h.enhancedServer,
			Notifications:  h.notifications,
			Agones:         h.agones,
		}
		if err := cmd.Execute(ctx); err != nil {
			log.Printf("Slash command error: %v", err)
			_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: strPtr("❌ An error occurred while executing the command.")})
			return
		}
		// Provide a simple success ack if command did not send anything
		_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: strPtr("✅ Command executed.")})
	} else {
		_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: strPtr("❓ Unknown command.")})
	}
}

func strPtr(s string) *string { return &s }
