package discord

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"agis-bot/internal/agones"
)

// RegisterAgonesCommands registers all Agones-related Discord commands
func (b *Bot) RegisterAgonesCommands() {
	b.RegisterCommand("!servers", b.handleListServers, "List all game servers")
	b.RegisterCommand("!fleet", b.handleFleetStatus, "Show fleet status")
	b.RegisterCommand("!allocate", b.handleAllocateServer, "Allocate a game server")
	b.RegisterCommand("!scale", b.handleScaleFleet, "Scale fleet (admin only)")
	b.RegisterCommand("!create", b.handleCreateServer, "Create a new game server")
	b.RegisterCommand("!delete", b.handleDeleteServer, "Delete a game server")
	b.RegisterCommand("!status", b.handleServerStatus, "Show detailed server status")
}

// handleListServers lists all game servers
func (b *Bot) handleListServers(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	status, err := b.agonesClient.GetGameServerStatus(ctx)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("❌ Error getting game servers: %v", err))
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:       "🎮 Game Servers",
		Description: fmt.Sprintf("Total: %d", status.Total),
		Color:       0x00ff00,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Ready", Value: fmt.Sprintf("%d", status.Ready), Inline: true},
			{Name: "Allocated", Value: fmt.Sprintf("%d", status.Allocated), Inline: true},
			{Name: "Error", Value: fmt.Sprintf("%d", status.Error), Inline: true},
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	// Add server list (limit to 10 for Discord message limits)
	if len(status.GameServers) > 0 {
		var serverList []string
		for i, gs := range status.GameServers {
			if i >= 10 {
				serverList = append(serverList, fmt.Sprintf("... and %d more", len(status.GameServers)-10))
				break
			}
			emoji := getStateEmoji(gs.State)
			serverList = append(serverList, fmt.Sprintf("%s **%s** - %s:%d", 
				emoji, gs.Name, gs.Address, gs.Port))
		}
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:  "Servers",
			Value: strings.Join(serverList, "\n"),
		})
	}

	s.ChannelMessageSendEmbed(m.ChannelID, embed)
}

// handleFleetStatus shows fleet status
func (b *Bot) handleFleetStatus(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fleetName := "agis-dev-fleet"
	if len(args) > 0 {
		fleetName = args[0]
	}

	fleet, err := b.agonesClient.GetFleet(ctx, fleetName)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("❌ Error getting fleet: %v", err))
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("⚓ Fleet: %s", fleet.Name),
		Description: "Fleet Status",
		Color:       0x0099ff,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Replicas", Value: fmt.Sprintf("%d", fleet.Spec.Replicas), Inline: true},
			{Name: "Ready", Value: fmt.Sprintf("%d", fleet.Status.ReadyReplicas), Inline: true},
			{Name: "Allocated", Value: fmt.Sprintf("%d", fleet.Status.AllocatedReplicas), Inline: true},
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	s.ChannelMessageSendEmbed(m.ChannelID, embed)
}

// handleAllocateServer allocates a game server from the fleet
func (b *Bot) handleAllocateServer(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fleetName := "agis-dev-fleet"
	if len(args) > 0 {
		fleetName = args[0]
	}

	allocation, err := b.agonesClient.AllocateGameServer(ctx, fleetName)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("❌ Error allocating server: %v", err))
		return
	}

	if allocation.Status.State == "Allocated" {
		embed := &discordgo.MessageEmbed{
			Title:       "✅ Server Allocated",
			Description: fmt.Sprintf("Successfully allocated server from fleet %s", fleetName),
			Color:       0x00ff00,
			Fields: []*discordgo.MessageEmbedField{
				{Name: "Server", Value: allocation.Status.GameServerName, Inline: true},
				{Name: "Address", Value: allocation.Status.Address, Inline: true},
				{Name: "Port", Value: fmt.Sprintf("%d", allocation.Status.Ports[0].Port), Inline: true},
			},
			Timestamp: time.Now().Format(time.RFC3339),
		}
		s.ChannelMessageSendEmbed(m.ChannelID, embed)
	} else {
		s.ChannelMessageSend(m.ChannelID, "⚠️ Server allocation failed - no servers available")
	}
}

// handleScaleFleet scales the fleet (admin only)
func (b *Bot) handleScaleFleet(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	// Check if user has admin role
	if !b.hasAdminRole(s, m.GuildID, m.Author.ID) {
		s.ChannelMessageSend(m.ChannelID, "❌ You need admin permissions to scale fleets")
		return
	}

	if len(args) < 2 {
		s.ChannelMessageSend(m.ChannelID, "Usage: !scale <fleet-name> <replicas>")
		return
	}

	fleetName := args[0]
	var replicas int32
	fmt.Sscanf(args[1], "%d", &replicas)

	if replicas < 0 || replicas > 20 {
		s.ChannelMessageSend(m.ChannelID, "❌ Replicas must be between 0 and 20")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := b.agonesClient.ScaleFleet(ctx, fleetName, replicas)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("❌ Error scaling fleet: %v", err))
		return
	}

	s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("✅ Fleet %s scaled to %d replicas", fleetName, replicas))
}

// handleCreateServer creates a new game server
func (b *Bot) handleCreateServer(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	name := "test-server"
	if len(args) > 0 {
		name = args[0]
	}

	gs, err := b.agonesClient.CreateSimpleGameServer(ctx, name)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("❌ Error creating server: %v", err))
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:       "✅ Server Created",
		Description: "New game server created successfully",
		Color:       0x00ff00,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Name", Value: gs.Name, Inline: true},
			{Name: "State", Value: string(gs.Status.State), Inline: true},
			{Name: "Node", Value: gs.Status.NodeName, Inline: true},
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	s.ChannelMessageSendEmbed(m.ChannelID, embed)
}

// handleDeleteServer deletes a game server
func (b *Bot) handleDeleteServer(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	if len(args) < 1 {
		s.ChannelMessageSend(m.ChannelID, "Usage: !delete <server-name>")
		return
	}

	serverName := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := b.agonesClient.DeleteGameServer(ctx, serverName)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("❌ Error deleting server: %v", err))
		return
	}

	s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("✅ Server %s deleted", serverName))
}

// handleServerStatus shows detailed server status
func (b *Bot) handleServerStatus(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	if len(args) < 1 {
		// Show overall status
		b.handleListServers(s, m, args)
		return
	}

	serverName := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	gs, err := b.agonesClient.GetGameServer(ctx, serverName)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("❌ Error getting server: %v", err))
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("🎮 %s", gs.Name),
		Description: "Game Server Details",
		Color:       getStateColor(string(gs.Status.State)),
		Fields: []*discordgo.MessageEmbedField{
			{Name: "State", Value: string(gs.Status.State), Inline: true},
			{Name: "Address", Value: gs.Status.Address, Inline: true},
			{Name: "Node", Value: gs.Status.NodeName, Inline: true},
		},
		Timestamp: gs.CreationTimestamp.Format(time.RFC3339),
	}

	if len(gs.Status.Ports) > 0 {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Port",
			Value:  fmt.Sprintf("%d", gs.Status.Ports[0].Port),
			Inline: true,
		})
	}

	s.ChannelMessageSendEmbed(m.ChannelID, embed)
}

// hasAdminRole checks if user has admin role
func (b *Bot) hasAdminRole(s *discordgo.Session, guildID, userID string) bool {
	member, err := s.GuildMember(guildID, userID)
	if err != nil {
		return false
	}

	guild, err := s.Guild(guildID)
	if err != nil {
		return false
	}

	adminRoles := []string{"Admin", "GameMaster", "Administrator"}

	for _, roleID := range member.Roles {
		for _, role := range guild.Roles {
			if role.ID == roleID {
				for _, adminRole := range adminRoles {
					if strings.EqualFold(role.Name, adminRole) {
						return true
					}
				}
			}
		}
	}

	return false
}

// Helper functions

func getStateEmoji(state string) string {
	switch state {
	case "Ready":
		return "🟢"
	case "Allocated":
		return "🔵"
	case "Reserved":
		return "🟡"
	case "Shutdown":
		return "⚫"
	case "Error":
		return "🔴"
	default:
		return "⚪"
	}
}

func getStateColor(state string) int {
	switch state {
	case "Ready":
		return 0x00ff00 // Green
	case "Allocated":
		return 0x0099ff // Blue
	case "Reserved":
		return 0xffff00 // Yellow
	case "Shutdown":
		return 0x808080 // Gray
	case "Error":
		return 0xff0000 // Red
	default:
		return 0xffffff // White
	}
}