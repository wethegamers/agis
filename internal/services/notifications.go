package services

import (
	"fmt"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
)

// NotificationService handles sending notifications to users about server status
type NotificationService struct {
	discord *discordgo.Session
	db      *DatabaseService
	logging *LoggingService
}

// ServerStatusUpdate represents a server status change
type ServerStatusUpdate struct {
	ServerName     string
	UserID         string
	PreviousStatus string
	NewStatus      string
	Address        string
	Port           int32
	GameType       string
	ErrorMessage   string
	ChannelID      string // If provided, send notification to this channel instead of DM
}

// NewNotificationService creates a new notification service
func NewNotificationService(discord *discordgo.Session, db *DatabaseService, logging *LoggingService) *NotificationService {
	return &NotificationService{
		discord: discord,
		db:      db,
		logging: logging,
	}
}

// SetDiscordSession sets the Discord session for the notification service
func (n *NotificationService) SetDiscordSession(session *discordgo.Session) {
	n.discord = session
}

// NotifyServerStatusChange sends a notification to the user about server status changes
func (n *NotificationService) NotifyServerStatusChange(update ServerStatusUpdate) error {
	var targetChannelID string
	var channelType string
	
	// Determine target channel - use provided channel ID or create DM
	if update.ChannelID != "" {
		targetChannelID = update.ChannelID
		channelType = "channel"
	} else {
		// Create DM channel with user
		channel, err := n.discord.UserChannelCreate(update.UserID)
		if err != nil {
			log.Printf("Failed to create DM channel for user %s: %v", update.UserID, err)
			return err
		}
		targetChannelID = channel.ID
		channelType = "DM"
	}

	var embed *discordgo.MessageEmbed

	switch update.NewStatus {
	case "Pending":
		embed = n.createPendingEmbed(update)
	case "Creating":
		embed = n.createCreatingEmbed(update)
	case "Starting":
		embed = n.createStartingEmbed(update)
	case "Ready", "Allocated":
		embed = n.createReadyEmbed(update)
	case "Error", "Failed":
		embed = n.createErrorEmbed(update)
	case "Shutdown":
		embed = n.createShutdownEmbed(update)
	default:
		embed = n.createGenericUpdateEmbed(update)
	}

	_, err := n.discord.ChannelMessageSendEmbed(targetChannelID, embed)
	if err != nil {
		log.Printf("Failed to send %s notification to user %s: %v", channelType, update.UserID, err)
		return err
	}

	// Log the notification
	if n.logging != nil {
		log.Printf("Sent status notification to user %s for server %s via %s: %s -> %s", 
			update.UserID, update.ServerName, channelType, update.PreviousStatus, update.NewStatus)
	}

	return nil
}

func (n *NotificationService) createPendingEmbed(update ServerStatusUpdate) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       "🎮 Server Deployment Started",
		Description: fmt.Sprintf("Your **%s** server **%s** is being prepared for deployment.", update.GameType, update.ServerName),
		Color:       0x3498db,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "⏱️ Status",
				Value:  "⚡ Requesting resources from the cluster",
				Inline: false,
			},
			{
				Name:   "📋 Next Steps",
				Value:  "• Container image will be pulled\n• Pod will be scheduled on a node\n• Game server will initialize",
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Estimated time: 2-3 minutes • You'll be notified of each step",
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}
}

func (n *NotificationService) createCreatingEmbed(update ServerStatusUpdate) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       "🚀 Server Container Starting",
		Description: fmt.Sprintf("Your **%s** server **%s** container is now starting.", update.GameType, update.ServerName),
		Color:       0x3498db,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "⏱️ Status",
				Value:  "🔄 Container is being created and initialized",
				Inline: false,
			},
			{
				Name:   "📋 Progress",
				Value:  "✅ Resources allocated\n🔄 Container starting\n⏸️ Game server initializing\n⏸️ Health checks",
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Almost ready! Game initialization starting soon...",
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}
}

func (n *NotificationService) createStartingEmbed(update ServerStatusUpdate) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       "⚡ Game Server Initializing",
		Description: fmt.Sprintf("Your **%s** server **%s** is initializing the game world.", update.GameType, update.ServerName),
		Color:       0xf39c12,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "⏱️ Status",
				Value:  "🎯 Game server is loading and preparing for connections",
				Inline: false,
			},
			{
				Name:   "📋 Progress",
				Value:  "✅ Resources allocated\n✅ Container started\n🔄 Game server initializing\n⏸️ Health checks",
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Final step! Connection details coming shortly...",
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}
}

func (n *NotificationService) createReadyEmbed(update ServerStatusUpdate) *discordgo.MessageEmbed {
	embed := &discordgo.MessageEmbed{
		Title:       "🎉 Server Ready to Play!",
		Description: fmt.Sprintf("Your **%s** server **%s** is now online and ready for connections!", update.GameType, update.ServerName),
		Color:       0x27ae60,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "⏱️ Status",
				Value:  "✅ **ONLINE** - Ready for players",
				Inline: false,
			},
			{
				Name:   "📋 Progress",
				Value:  "✅ Resources allocated\n✅ Container started\n✅ Game server initialized\n✅ Health checks passed",
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "🎮 Have fun playing! Use 'servers' to see all your servers",
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	// Add connection info if available
	if update.Address != "" && update.Port > 0 {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "🌐 Connection Details",
			Value:  fmt.Sprintf("**Server Address:** `%s:%d`\n**Game Type:** %s", update.Address, update.Port, update.GameType),
			Inline: false,
		})

		// Add game-specific connection instructions
		switch update.GameType {
		case "minecraft":
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:   "🎮 How to Connect",
				Value:  fmt.Sprintf("1. Open Minecraft Java Edition\n2. Go to Multiplayer\n3. Add Server: `%s:%d`\n4. Join and start playing!", update.Address, update.Port),
				Inline: false,
			})
		case "cs2":
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:   "🎮 How to Connect",
				Value:  fmt.Sprintf("1. Open CS2\n2. Press ~ to open console\n3. Type: `connect %s:%d`\n4. Press Enter and enjoy!", update.Address, update.Port),
				Inline: false,
			})
		case "terraria":
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:   "🎮 How to Connect",
				Value:  fmt.Sprintf("1. Open Terraria\n2. Go to Multiplayer\n3. Join via IP: `%s:%d`\n4. Start your adventure!", update.Address, update.Port),
				Inline: false,
			})
		}
	}

	return embed
}

func (n *NotificationService) createErrorEmbed(update ServerStatusUpdate) *discordgo.MessageEmbed {
	embed := &discordgo.MessageEmbed{
		Title:       "❌ Server Deployment Failed",
		Description: fmt.Sprintf("Unfortunately, your **%s** server **%s** encountered an error during deployment.", update.GameType, update.ServerName),
		Color:       0xe74c3c,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "⏱️ Status",
				Value:  "❌ **FAILED** - Deployment unsuccessful",
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "🆘 Try again or contact support if this persists",
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	if update.ErrorMessage != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "🔍 Error Details",
			Value:  fmt.Sprintf("```%s```", update.ErrorMessage),
			Inline: false,
		})
	}

	embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
		Name:   "🛠️ What to do next",
		Value:  "• Try creating the server again\n• Check your credits balance\n• Use `diagnostics " + update.ServerName + "` for details\n• Contact support if the issue persists",
		Inline: false,
	})

	return embed
}

func (n *NotificationService) createShutdownEmbed(update ServerStatusUpdate) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       "⏹️ Server Shutdown",
		Description: fmt.Sprintf("Your **%s** server **%s** has been shut down.", update.GameType, update.ServerName),
		Color:       0x95a5a6,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "⏱️ Status",
				Value:  "⏹️ **STOPPED** - No longer consuming credits",
				Inline: false,
			},
			{
				Name:   "💾 Data Preservation",
				Value:  "Your world/save data has been preserved and can be restored when you create a new server.",
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Create a new server anytime with the 'create' command",
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}
}

func (n *NotificationService) createGenericUpdateEmbed(update ServerStatusUpdate) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       "📢 Server Status Update",
		Description: fmt.Sprintf("Your **%s** server **%s** status has changed.", update.GameType, update.ServerName),
		Color:       0x3498db,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Status Change",
				Value:  fmt.Sprintf("**From:** %s\n**To:** %s", update.PreviousStatus, update.NewStatus),
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Use 'diagnostics " + update.ServerName + "' for detailed information",
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}
}

// NotifyServerReady is a convenience method for when a server becomes ready
func (n *NotificationService) NotifyServerReady(userID, serverName, gameType, address string, port int32) error {
	return n.NotifyServerStatusChange(ServerStatusUpdate{
		ServerName:     serverName,
		UserID:         userID,
		PreviousStatus: "Starting",
		NewStatus:      "Ready",
		Address:        address,
		Port:           port,
		GameType:       gameType,
	})
}

// NotifyServerError is a convenience method for when a server encounters an error
func (n *NotificationService) NotifyServerError(userID, serverName, gameType, errorMessage string) error {
	return n.NotifyServerStatusChange(ServerStatusUpdate{
		ServerName:     serverName,
		UserID:         userID,
		PreviousStatus: "Creating",
		NewStatus:      "Error",
		GameType:       gameType,
		ErrorMessage:   errorMessage,
	})
}

// NotifyServerErrorInChannel is a convenience method for when a server encounters an error with channel notification
func (n *NotificationService) NotifyServerErrorInChannel(userID, serverName, gameType, errorMessage, channelID string) error {
	return n.NotifyServerStatusChange(ServerStatusUpdate{
		ServerName:     serverName,
		UserID:         userID,
		PreviousStatus: "Creating",
		NewStatus:      "Error",
		GameType:       gameType,
		ErrorMessage:   errorMessage,
		ChannelID:      channelID,
	})
}
