package commands

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"agis-bot/internal/bot"

	"github.com/bwmarrin/discordgo"
)

type OwnerCommand struct{}

// updateEnvFile updates a key-value pair in the .env file for persistence
// In containerized environments, this gracefully handles missing .env files
func (c *OwnerCommand) updateEnvFile(key, value string) error {
	envPath := ".env"

	// Check if we're running in a containerized environment (Kubernetes)
	if os.Getenv("KUBERNETES_SERVICE_HOST") != "" {
		// Running in Kubernetes - skip .env file persistence
		// Configuration is handled via environment variables and in-memory state
		return nil
	}

	// Read the current .env file
	file, err := os.Open(envPath)
	if err != nil {
		// If .env file doesn't exist, create it
		if os.IsNotExist(err) {
			return os.WriteFile(envPath, []byte(fmt.Sprintf("%s=%s\n", key, value)), 0644)
		}
		return fmt.Errorf("failed to open .env file: %v", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			// Log the error but don't override the main error
			fmt.Printf("Warning: failed to close .env file: %v\n", closeErr)
		}
	}()

	var lines []string
	scanner := bufio.NewScanner(file)
	keyFound := false

	// Read all lines and update the target key
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, key+"=") {
			lines = append(lines, fmt.Sprintf("%s=%s", key, value))
			keyFound = true
		} else {
			lines = append(lines, line)
		}
	}

	// If key wasn't found, add it
	if !keyFound {
		lines = append(lines, fmt.Sprintf("%s=%s", key, value))
	}

	// Write back to the file
	return os.WriteFile(envPath, []byte(strings.Join(lines, "\n")+"\n"), 0644)
}

func (c *OwnerCommand) Name() string {
	return "owner"
}

func (c *OwnerCommand) Description() string {
	return "Owner-only commands for bot configuration"
}

func (c *OwnerCommand) RequiredPermission() bot.Permission {
	return bot.PermissionOwner
}

func (c *OwnerCommand) Execute(ctx *CommandContext) error {
	if len(ctx.Args) < 1 {
		return c.showHelp(ctx)
	}

	subcommand := strings.ToLower(ctx.Args[0])
	switch subcommand {
	case "set-admin":
		return c.setAdminRole(ctx, ctx.Args[1:])
	case "set-mod":
		return c.setModRole(ctx, ctx.Args[1:])
	case "list-roles":
		return c.listRoles(ctx)
	case "remove-admin":
		return c.removeAdminRole(ctx, ctx.Args[1:])
	case "remove-mod":
		return c.removeModRole(ctx, ctx.Args[1:])
	default:
		return c.showHelp(ctx)
	}
}

func (c *OwnerCommand) showHelp(ctx *CommandContext) error {
	embed := &discordgo.MessageEmbed{
		Title:       "👑 Bot Owner Commands",
		Description: "Configure bot permissions and roles",
		Color:       0xff6b6b,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "**Role Management**",
				Value:  "`owner set-admin <@role>` - Add admin role\n`owner set-mod <@role>` - Add moderator role\n`owner remove-admin <@role>` - Remove admin role\n`owner remove-mod <@role>` - Remove moderator role",
				Inline: false,
			},
			{
				Name:   "**Information**",
				Value:  "`owner list-roles` - Show current admin/mod roles",
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "⚠️ Owner-only commands • Bot Owner: You",
		},
	}

	_, err := ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	return err
}

func sendErrorEmbed(s *discordgo.Session, channelID, title, description string) error {
	embed := &discordgo.MessageEmbed{
		Title:       "❌ " + title,
		Description: description,
		Color:       0xff0000,
	}
	_, err := s.ChannelMessageSendEmbed(channelID, embed)
	return err
}

func (c *OwnerCommand) setAdminRole(ctx *CommandContext, args []string) error {
	if len(args) == 0 {
		return sendErrorEmbed(ctx.Session, ctx.Message.ChannelID, "Missing role", "Please mention a role to add as admin.")
	}

	// Extract role ID from mention
	roleID := extractRoleID(args[0])
	if roleID == "" {
		return sendErrorEmbed(ctx.Session, ctx.Message.ChannelID, "Invalid role", fmt.Sprintf("Please mention a valid role (e.g., @AdminRole). Got: %s", args[0]))
	}

	// Debug: Log what we're trying to look up
	fmt.Printf("Looking up role ID: %s in guild: %s\n", roleID, ctx.Message.GuildID)

	// Try to get role information via API first (more reliable than state)
	role, err := ctx.Session.State.Role(ctx.Message.GuildID, roleID)
	if err != nil {
		// Fallback to direct API call
		guild, guildErr := ctx.Session.Guild(ctx.Message.GuildID)
		if guildErr != nil {
			return sendErrorEmbed(ctx.Session, ctx.Message.ChannelID, "Guild error", fmt.Sprintf("Could not access guild information: %v", guildErr))
		}

		// Search for role in guild
		var foundRole *discordgo.Role
		for _, r := range guild.Roles {
			if r.ID == roleID {
				foundRole = r
				break
			}
		}

		if foundRole == nil {
			return sendErrorEmbed(ctx.Session, ctx.Message.ChannelID, "Role not found", fmt.Sprintf("Could not find role with ID %s in this server. Make sure you're mentioning a valid role.", roleID))
		}
		role = foundRole
	}

	// Add role to database
	err = ctx.DB.AddBotRole(roleID, "admin", ctx.Message.GuildID)
	if err != nil {
		return sendErrorEmbed(ctx.Session, ctx.Message.ChannelID, "Database Error", fmt.Sprintf("Failed to save admin role to database: %v", err))
	}

	// Update the bot's configuration in memory
	ctx.Config.Roles.AdminRoles = append(ctx.Config.Roles.AdminRoles, roleID)

	embed := &discordgo.MessageEmbed{
		Title:       "✅ Admin Role Added",
		Description: fmt.Sprintf("Role **%s** has been added to admin roles.", role.Name),
		Color:       0x00ff00,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Role ID",
				Value:  roleID,
				Inline: true,
			},
			{
				Name:   "Permissions",
				Value:  "• Full bot access\n• Backend cluster commands\n• All moderator abilities",
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "✅ Role configuration saved to database permanently",
		},
	}

	_, err = ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	return err
}

func (c *OwnerCommand) setModRole(ctx *CommandContext, args []string) error {
	if len(args) == 0 {
		return sendErrorEmbed(ctx.Session, ctx.Message.ChannelID, "Missing role", "Please mention a role to add as moderator.")
	}

	// Extract role ID from mention
	roleID := extractRoleID(args[0])
	if roleID == "" {
		return sendErrorEmbed(ctx.Session, ctx.Message.ChannelID, "Invalid role", fmt.Sprintf("Please mention a valid role (e.g., @ModRole). Got: %s", args[0]))
	}

	// Try to get role information via API first (more reliable than state)
	role, err := ctx.Session.State.Role(ctx.Message.GuildID, roleID)
	if err != nil {
		// Fallback to direct API call
		guild, guildErr := ctx.Session.Guild(ctx.Message.GuildID)
		if guildErr != nil {
			return sendErrorEmbed(ctx.Session, ctx.Message.ChannelID, "Guild error", fmt.Sprintf("Could not access guild information: %v", guildErr))
		}

		// Search for role in guild
		var foundRole *discordgo.Role
		for _, r := range guild.Roles {
			if r.ID == roleID {
				foundRole = r
				break
			}
		}

		if foundRole == nil {
			return sendErrorEmbed(ctx.Session, ctx.Message.ChannelID, "Role not found", fmt.Sprintf("Could not find role with ID %s in this server. Make sure you're mentioning a valid role.", roleID))
		}
		role = foundRole
	}

	// Add role to database
	err = ctx.DB.AddBotRole(roleID, "moderator", ctx.Message.GuildID)
	if err != nil {
		return sendErrorEmbed(ctx.Session, ctx.Message.ChannelID, "Database Error", fmt.Sprintf("Failed to save moderator role to database: %v", err))
	}

	// Update the bot's configuration in memory
	ctx.Config.Roles.ModRoles = append(ctx.Config.Roles.ModRoles, roleID)

	embed := &discordgo.MessageEmbed{
		Title:       "✅ Moderator Role Added",
		Description: fmt.Sprintf("Role **%s** has been added to moderator roles.", role.Name),
		Color:       0x00ff00,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Role ID",
				Value:  roleID,
				Inline: true,
			},
			{
				Name:   "Permissions",
				Value:  "• View all user servers\n• Control any user's server\n• Support commands",
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "✅ Role configuration saved to database permanently",
		},
	}

	_, err = ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	return err
}

func (c *OwnerCommand) listRoles(ctx *CommandContext) error {
	// Get roles from database
	adminRoleIDs, modRoleIDs, err := ctx.DB.GetAllBotRoles(ctx.Message.GuildID)
	if err != nil {
		return sendErrorEmbed(ctx.Session, ctx.Message.ChannelID, "Database Error", fmt.Sprintf("Failed to retrieve roles from database: %v", err))
	}

	adminRoles := "None configured"
	modRoles := "None configured"

	if len(adminRoleIDs) > 0 {
		var adminRoleStrings []string
		for _, roleID := range adminRoleIDs {
			if role, err := ctx.Session.State.Role(ctx.Message.GuildID, roleID); err == nil {
				adminRoleStrings = append(adminRoleStrings, fmt.Sprintf("<@&%s> (%s)", roleID, role.Name))
			} else {
				adminRoleStrings = append(adminRoleStrings, fmt.Sprintf("<@&%s>", roleID))
			}
		}
		adminRoles = strings.Join(adminRoleStrings, "\n")
	}

	if len(modRoleIDs) > 0 {
		var modRoleStrings []string
		for _, roleID := range modRoleIDs {
			if role, err := ctx.Session.State.Role(ctx.Message.GuildID, roleID); err == nil {
				modRoleStrings = append(modRoleStrings, fmt.Sprintf("<@&%s> (%s)", roleID, role.Name))
			} else {
				modRoleStrings = append(modRoleStrings, fmt.Sprintf("<@&%s>", roleID))
			}
		}
		modRoles = strings.Join(modRoleStrings, "\n")
	}

	embed := &discordgo.MessageEmbed{
		Title:       "🔐 Current Bot Roles",
		Description: "Currently configured admin and moderator roles (from database)",
		Color:       0x4169e1,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "**👑 Owner**",
				Value:  fmt.Sprintf("<@%s>", bot.BotOwnerID),
				Inline: false,
			},
			{
				Name:   "**⚙️ Admin Roles**",
				Value:  adminRoles,
				Inline: false,
			},
			{
				Name:   "**🛡️ Moderator Roles**",
				Value:  modRoles,
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Use 'owner set-admin' or 'owner set-mod' to configure roles",
		},
	}

	_, err = ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	return err
}

func (c *OwnerCommand) removeAdminRole(ctx *CommandContext, args []string) error {
	if len(args) == 0 {
		return sendErrorEmbed(ctx.Session, ctx.Message.ChannelID, "Missing role", "Please mention a role to remove from admin.")
	}

	roleID := extractRoleID(args[0])
	if roleID == "" {
		return sendErrorEmbed(ctx.Session, ctx.Message.ChannelID, "Invalid role", "Please mention a valid role (e.g., @AdminRole).")
	}

	// Remove role from database
	err := ctx.DB.RemoveBotRole(roleID, ctx.Message.GuildID)
	if err != nil {
		return sendErrorEmbed(ctx.Session, ctx.Message.ChannelID, "Database Error", fmt.Sprintf("Failed to remove admin role from database: %v", err))
	}

	// Update the bot's configuration in memory by removing the role
	var newAdminRoles []string
	for _, existingRoleID := range ctx.Config.Roles.AdminRoles {
		if existingRoleID != roleID {
			newAdminRoles = append(newAdminRoles, existingRoleID)
		}
	}
	ctx.Config.Roles.AdminRoles = newAdminRoles

	embed := &discordgo.MessageEmbed{
		Title:       "✅ Admin Role Removed",
		Description: "Role has been removed from admin roles.",
		Color:       0xffa500,
		Footer: &discordgo.MessageEmbedFooter{
			Text: "✅ Role configuration updated in database permanently",
		},
	}

	_, err = ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	return err
}

func (c *OwnerCommand) removeModRole(ctx *CommandContext, args []string) error {
	if len(args) == 0 {
		return sendErrorEmbed(ctx.Session, ctx.Message.ChannelID, "Missing role", "Please mention a role to remove from moderator.")
	}

	roleID := extractRoleID(args[0])
	if roleID == "" {
		return sendErrorEmbed(ctx.Session, ctx.Message.ChannelID, "Invalid role", "Please mention a valid role (e.g., @ModRole).")
	}

	// Remove role from database
	err := ctx.DB.RemoveBotRole(roleID, ctx.Message.GuildID)
	if err != nil {
		return sendErrorEmbed(ctx.Session, ctx.Message.ChannelID, "Database Error", fmt.Sprintf("Failed to remove moderator role from database: %v", err))
	}

	// Update the bot's configuration in memory by removing the role
	var newModRoles []string
	for _, existingRoleID := range ctx.Config.Roles.ModRoles {
		if existingRoleID != roleID {
			newModRoles = append(newModRoles, existingRoleID)
		}
	}
	ctx.Config.Roles.ModRoles = newModRoles

	embed := &discordgo.MessageEmbed{
		Title:       "✅ Moderator Role Removed",
		Description: "Role has been removed from moderator roles.",
		Color:       0xffa500,
		Footer: &discordgo.MessageEmbedFooter{
			Text: "✅ Role configuration updated in database permanently",
		},
	}

	_, err = ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	return err
}

// extractRoleID extracts role ID from Discord role mention
func extractRoleID(mention string) string {
	// Discord role mentions are in format <@&ROLE_ID>
	if strings.HasPrefix(mention, "<@&") && strings.HasSuffix(mention, ">") {
		return mention[3 : len(mention)-1]
	}
	// Handle alternate format @&ROLE_ID (without < >)
	if strings.HasPrefix(mention, "@&") {
		return mention[2:]
	}
	// If it's just a raw ID (numeric string)
	if len(mention) >= 17 && len(mention) <= 20 {
		// Basic check if it's all digits
		for _, r := range mention {
			if r < '0' || r > '9' {
				return ""
			}
		}
		return mention
	}
	return ""
}

// removeRoleID removes a role ID from a comma-separated list of role IDs
func removeRoleID(roleList, roleID string) string {
	roleIDs := strings.Split(roleList, ",")
	var newRoleIDs []string
	for _, id := range roleIDs {
		if id != roleID {
			newRoleIDs = append(newRoleIDs, id)
		}
	}
	return strings.Join(newRoleIDs, ",")
}
