package commands

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"agis-bot/internal/bot"
	"agis-bot/internal/config"
	"agis-bot/internal/services"

	"github.com/bwmarrin/discordgo"
)

// CommandContext holds the context for command execution
type CommandContext struct {
	Session        *discordgo.Session
	Message        *discordgo.MessageCreate
	Args           []string
	DB             *services.DatabaseService
	Config         *config.Config
	Permissions    *bot.PermissionChecker
	UserPerm       bot.Permission
	Logger         *services.LoggingService
	Context        context.Context
	EnhancedServer *services.EnhancedServerService
	Notifications  *services.NotificationService
	Agones         *services.AgonesService
}

// Command represents a bot command
type Command interface {
	Name() string
	Description() string
	RequiredPermission() bot.Permission
	Execute(ctx *CommandContext) error
}

// CommandHandler manages all bot commands
type CommandHandler struct {
	commands       map[string]Command
	config         *config.Config
	db             *services.DatabaseService
	permissions    *bot.PermissionChecker
	logger         *services.LoggingService
	enhancedServer *services.EnhancedServerService
	notifications  *services.NotificationService
	agones         *services.AgonesService
}

func NewCommandHandler(cfg *config.Config, db *services.DatabaseService, logger *services.LoggingService) *CommandHandler {
	// Initialize Agones service
	agonesService, err := services.NewAgonesService()
	if err != nil {
		log.Printf("⚠️ Failed to initialize Agones service: %v", err)
		agonesService = nil
	}

	// Initialize notification service
	notificationService := services.NewNotificationService(nil, db, logger) // Session will be set later

	// Initialize enhanced server service
	var enhancedService *services.EnhancedServerService
	if agonesService != nil {
		enhancedService = services.NewEnhancedServerService(db, agonesService, notificationService)
	}

	handler := &CommandHandler{
		commands:       make(map[string]Command),
		config:         cfg,
		db:             db,
		permissions:    bot.NewPermissionChecker(cfg),
		logger:         logger,
		enhancedServer: enhancedService,
		notifications:  notificationService,
		agones:         agonesService,
	}

	// Register all commands
	handler.registerCommands()
	return handler
}

func (h *CommandHandler) registerCommands() {
	// User commands
	h.Register(&HelpCommand{})
	h.Register(&ManualCommand{})
	h.Register(&ManCommand{})
	h.Register(&CreditsCommand{})
	h.Register(&CreditsEarnCommand{})
	h.Register(&DailyCommand{})
	h.Register(&WorkCommand{})
	h.Register(&ServersCommand{})
	h.Register(&CreateServerCommand{})
	h.Register(&StopServerCommand{})
	h.Register(&DeleteServerCommand{})
	h.Register(&ConfirmDeleteMineCommand{})
	h.Register(NewExportSaveCommand())
	h.Register(&PublicLobbyCommand{})
	h.Register(&DiagnosticsCommand{})
	h.Register(&PingCommand{})
	
	// v1.3.0 New commands
	h.Register(&RestartServerCommand{})
	h.Register(&StartServerCommand{})
	h.Register(&ServerLogsCommand{})
	h.Register(&ProfileCommand{})
	h.Register(NewInfoAboutCommand(time.Now())) // Pass bot start time
	h.Register(&InfoGamesCommand{})
	h.Register(&LeaderboardCommand{})
	
	// v1.4.0 Medium priority commands
	h.Register(&GiftCreditsCommand{})
	h.Register(&TransactionsCommand{})
	h.Register(&FavoriteCommand{})
	h.Register(&SearchServersCommand{})
	h.Register(&ShopCommand{})
	
	// v1.5.0 Low priority commands
	h.Register(&AchievementsCommand{})
	h.Register(&ReviewCommand{})
	h.Register(&ReviewsCommand{})

	// Debug command
	h.Register(&DebugPermissionsCommand{})

	// Mod commands
	h.Register(&ModServersCommand{})
	h.Register(&ModControlCommand{})
	h.Register(&ModDeleteCommand{})
	h.Register(&ConfirmDeleteCommand{})

	// Admin commands
	h.Register(&AdminStatusCommand{})
	h.Register(&AdminRestartCommand{})
	h.Register(&LogChannelCommand{})
	h.Register(&AdoptCommand{})

	// Owner commands
	h.Register(&OwnerCommand{})
}

func (h *CommandHandler) Register(cmd Command) {
	h.commands[strings.ToLower(cmd.Name())] = cmd
}

// EnhancedService returns the EnhancedServerService (may be nil if Agones unavailable)
func (h *CommandHandler) EnhancedService() *services.EnhancedServerService {
	return h.enhancedServer
}

func (h *CommandHandler) HandleMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore messages from bots
	if m.Author.Bot {
		return
	}

	// Only respond to mentions or DMs
	if m.GuildID != "" {
		altMention := fmt.Sprintf("<@!%s>", s.State.User.ID)
		if !(strings.Contains(m.Content, s.State.User.Mention()) || strings.Contains(m.Content, altMention)) {
			return
		}
	}

	content := strings.TrimSpace(m.Content)
	// Strip both mention formats <@id> and <@!id>
	content = strings.ReplaceAll(content, s.State.User.Mention(), "")
	altMention := fmt.Sprintf("<@!%s>", s.State.User.ID)
	content = strings.ReplaceAll(content, altMention, "")
	content = strings.ToLower(strings.TrimSpace(content))

	args := strings.Fields(content)
	if len(args) == 0 {
		// Show help if no command provided
		args = []string{"help"}
	}

	commandName := args[0]

	// Record command usage
	if h.db != nil {
		h.db.RecordCommandUsage(m.Author.ID, commandName)
	}

// Get user permission level
	userPerm := h.permissions.GetUserPermission(s, m.GuildID, m.Author.ID)

	// Enforce Verified role if configured (allow a minimal public set)
	if h.config != nil && h.config.Roles.VerifiedRoleID != "" && m.GuildID != "" {
		allowed := map[string]bool{
			"help":         true,
			"manual":       true,
			"man":          true,
			"credits":      true,
			"credits_earn": true,
			"ping":         true,
		}
		if !h.permissions.IsVerified(s, m.GuildID, m.Author.ID) && !allowed[commandName] {
			h.sendError(s, m, "You must be Verified to use this command. Visit the dashboard to request access.")
			return
		}
	}

	// Find and execute command
	if cmd, exists := h.commands[commandName]; exists {
		// Check permissions
		if userPerm < cmd.RequiredPermission() {
			h.sendPermissionDenied(s, m, cmd.RequiredPermission())
			return
		}

		// Create command context
		ctx := &CommandContext{
			Session:        s,
			Message:        m,
			Args:           args[1:], // Remove command name from args
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

		// Log command execution
		if h.logger != nil {
			h.logger.LogUser(m.Author.ID, "command_executed", fmt.Sprintf("User executed command: %s", commandName), map[string]interface{}{
				"command": commandName,
				"args":    len(args) - 1,
				"guild":   m.GuildID,
				"channel": m.ChannelID,
			})
		}

		// Execute command
		if err := cmd.Execute(ctx); err != nil {
			log.Printf("Command execution error: %v", err)

			// Log error
			if h.logger != nil {
				h.logger.LogError("command_error", fmt.Sprintf("Command execution failed: %s", commandName), map[string]interface{}{
					"command": commandName,
					"user":    m.Author.ID,
					"error":   err.Error(),
					"guild":   m.GuildID,
					"channel": m.ChannelID,
				})
			}

			h.sendError(s, m, "An error occurred while executing the command.")
		}
	} else {
		// Unknown command, show help
		if helpCmd, exists := h.commands["help"]; exists {
			ctx := &CommandContext{
				Session:        s,
				Message:        m,
				Args:           []string{},
				DB:             h.db,
				Config:         h.config,
				Permissions:    h.permissions,
				UserPerm:       userPerm,
				Context:        context.Background(),
				EnhancedServer: h.enhancedServer,
				Notifications:  h.notifications,
				Agones:         h.agones,
			}
			helpCmd.Execute(ctx)
		}
	}
}

func (h *CommandHandler) sendPermissionDenied(s *discordgo.Session, m *discordgo.MessageCreate, required bot.Permission) {
	embed := &discordgo.MessageEmbed{
		Title:       "❌ Permission Denied",
		Description: fmt.Sprintf("This command requires **%s** permissions.", bot.GetPermissionString(required)),
		Color:       0xff0000,
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Contact a moderator if you believe this is an error",
		},
	}
	s.ChannelMessageSendEmbed(m.ChannelID, embed)
}

func (h *CommandHandler) sendError(s *discordgo.Session, m *discordgo.MessageCreate, message string) {
	embed := &discordgo.MessageEmbed{
		Title:       "❌ Error",
		Description: message,
		Color:       0xff0000,
	}
	s.ChannelMessageSendEmbed(m.ChannelID, embed)
}

func (h *CommandHandler) GetCommands() map[string]Command {
	return h.commands
}

// SetDiscordSession sets the Discord session for services that need it
func (h *CommandHandler) SetDiscordSession(session *discordgo.Session) {
	if h.notifications != nil {
		h.notifications.SetDiscordSession(session)
	}
}
