package commands

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"agis-bot/internal/bot"
	"agis-bot/internal/services"

	"github.com/bwmarrin/discordgo"
)

// Utility: extract raw Discord user ID from mention formats <@123> or <@!123>
func parseDiscordID(mention string) string {
    mention = strings.TrimSpace(mention)
    re := regexp.MustCompile(`^<@!?([0-9]+)>$`)
    matches := re.FindStringSubmatch(mention)
    if len(matches) == 2 {
        return matches[1]
    }
    return mention // fallback: maybe already an ID
}

// Shared interface for guild-related commands requiring treasury service
type guildServiceAware interface {
    guildService() *services.GuildTreasuryService
}

// ----------------------------------------------------------------------------
// Guild Create Command (/guild create <name>)
// ----------------------------------------------------------------------------

type GuildCreateCommand struct { Service *services.GuildTreasuryService }

func NewGuildCreateCommand(s *services.GuildTreasuryService) *GuildCreateCommand { return &GuildCreateCommand{Service: s} }
func (c *GuildCreateCommand) Name() string        { return "guild-create" }
func (c *GuildCreateCommand) Description() string { return "Create a new guild treasury (owner only)" }
func (c *GuildCreateCommand) RequiredPermission() bot.Permission { return bot.PermissionUser }

func sanitizeGuildName(name string) (id string) {
    name = strings.TrimSpace(name)
    if len(name) > 32 { name = name[:32] }
    // ID: lowercase, spaces -> '-', drop invalid chars
    lower := strings.ToLower(name)
    re := regexp.MustCompile(`[^a-z0-9-]+`)
    id = strings.ReplaceAll(lower, " ", "-")
    id = re.ReplaceAllString(id, "")
    if id == "" { id = fmt.Sprintf("guild-%d", time.Now().Unix()) }
    return id
}

func (c *GuildCreateCommand) Execute(ctx *CommandContext) error {
    if ctx.DB == nil { return fmt.Errorf("database unavailable") }
    if len(ctx.Args) < 1 { return fmt.Errorf("usage: guild-create <guild-name>") }

    rawName := strings.Join(ctx.Args, " ")
    guildID := sanitizeGuildName(rawName)

    user, err := ctx.DB.GetOrCreateUser(ctx.Message.Author.ID)
    if err != nil { return fmt.Errorf("failed to load user: %v", err) }

    // Check if user already in a guild with same ID (optional future enhancement)
    svc := c.Service
    if svc == nil { return fmt.Errorf("guild treasury service unavailable") }

    if err := svc.CreateGuild(guildID, rawName, user.DiscordID); err != nil {
        embed := &discordgo.MessageEmbed{Title: "❌ Guild Creation Failed", Description: err.Error(), Color: 0xff0000}
        ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
        return nil
    }

    embed := &discordgo.MessageEmbed{
        Title:       "✅ Guild Created",
        Description: fmt.Sprintf("Guild **%s** (`%s`) created. Use `guild-invite <@user> %s` to add members.\nDeposits are non-refundable.", rawName, guildID, guildID),
        Color:       0x00ff66,
        Fields: []*discordgo.MessageEmbedField{
            {Name: "Owner", Value: fmt.Sprintf("<@%s>", user.DiscordID), Inline: true},
            {Name: "Guild ID", Value: guildID, Inline: true},
        },
        Footer: &discordgo.MessageEmbedFooter{Text: "Titan-tier servers require guild pooled funding."},
    }
    ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
    return nil
}

// ----------------------------------------------------------------------------
// Guild Invite Command (/guild invite <@user> <guild_id>)
// ----------------------------------------------------------------------------

type GuildInviteCommand struct { Service *services.GuildTreasuryService }

func NewGuildInviteCommand(s *services.GuildTreasuryService) *GuildInviteCommand { return &GuildInviteCommand{Service: s} }
func (c *GuildInviteCommand) Name() string        { return "guild-invite" }
func (c *GuildInviteCommand) Description() string { return "Invite a user to your guild (owner/admin)" }
func (c *GuildInviteCommand) RequiredPermission() bot.Permission { return bot.PermissionUser }

func (c *GuildInviteCommand) Execute(ctx *CommandContext) error {
    if len(ctx.Args) < 2 { return fmt.Errorf("usage: guild-invite <@user> <guild_id>") }
    targetID := parseDiscordID(ctx.Args[0])
    guildID := ctx.Args[1]

    if c.Service == nil { return fmt.Errorf("guild treasury service unavailable") }

    if err := c.Service.AddMember(guildID, targetID, ctx.Message.Author.ID); err != nil {
        embed := &discordgo.MessageEmbed{Title: "❌ Invite Failed", Description: err.Error(), Color: 0xff0000}
        ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
        return nil
    }

    embed := &discordgo.MessageEmbed{
        Title:       "👥 Member Invited",
        Description: fmt.Sprintf("<@%s> added to guild `%s`", targetID, guildID),
        Color:       0x3399ff,
    }
    ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
    return nil
}

// ----------------------------------------------------------------------------
// Guild Deposit Command (/guild deposit <guild_id> <amount>)
// ----------------------------------------------------------------------------

type GuildDepositCommand struct { Service *services.GuildTreasuryService }

func NewGuildDepositCommand(s *services.GuildTreasuryService) *GuildDepositCommand { return &GuildDepositCommand{Service: s} }
func (c *GuildDepositCommand) Name() string        { return "guild-deposit" }
func (c *GuildDepositCommand) Description() string { return "Deposit GameCredits into guild treasury" }
func (c *GuildDepositCommand) RequiredPermission() bot.Permission { return bot.PermissionUser }

func (c *GuildDepositCommand) Execute(ctx *CommandContext) error {
    if len(ctx.Args) < 2 { return fmt.Errorf("usage: guild-deposit <guild_id> <amount>") }
    guildID := ctx.Args[0]
    var amount int
    fmt.Sscanf(ctx.Args[1], "%d", &amount)
    if amount <= 0 { return fmt.Errorf("amount must be positive") }

    if c.Service == nil { return fmt.Errorf("guild treasury service unavailable") }

    if err := c.Service.DepositToGuild(guildID, ctx.Message.Author.ID, amount); err != nil {
        embed := &discordgo.MessageEmbed{Title: "❌ Deposit Failed", Description: err.Error(), Color: 0xff0000}
        ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
        return nil
    }

    guild, _ := c.Service.GetGuild(guildID)
    balanceStr := "(balance unavailable)"
    if guild != nil { balanceStr = fmt.Sprintf("New Balance: %d GC", guild.Balance) }

    embed := &discordgo.MessageEmbed{
        Title:       "💰 Deposit Successful",
        Description: fmt.Sprintf("%d GC deposited to guild `%s`. %s", amount, guildID, balanceStr),
        Color:       0x00ccff,
    }
    ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
    return nil
}

// ----------------------------------------------------------------------------
// Guild Treasury Command (/guild treasury <guild_id>) - view balance + members
// ----------------------------------------------------------------------------

type GuildTreasuryCommand struct { Service *services.GuildTreasuryService }

func NewGuildTreasuryCommand(s *services.GuildTreasuryService) *GuildTreasuryCommand { return &GuildTreasuryCommand{Service: s} }
func (c *GuildTreasuryCommand) Name() string        { return "guild-treasury" }
func (c *GuildTreasuryCommand) Description() string { return "View guild treasury balance and top contributors" }
func (c *GuildTreasuryCommand) RequiredPermission() bot.Permission { return bot.PermissionUser }

func (c *GuildTreasuryCommand) Execute(ctx *CommandContext) error {
    if len(ctx.Args) < 1 { return fmt.Errorf("usage: guild-treasury <guild_id>") }
    guildID := ctx.Args[0]
    if c.Service == nil { return fmt.Errorf("guild treasury service unavailable") }

    guild, err := c.Service.GetGuild(guildID)
    if err != nil {
        embed := &discordgo.MessageEmbed{Title: "❌ Guild Not Found", Description: err.Error(), Color: 0xff0000}
        ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
        return nil
    }

    members, _ := c.Service.GetGuildMembers(guildID)
    var topLines []string
    for i, m := range members {
        if i >= 5 { break }
        topLines = append(topLines, fmt.Sprintf("%d. <@%s> – %d GC", i+1, m.DiscordID, m.TotalDeposits))
    }
    if len(topLines) == 0 { topLines = []string{"No contributors yet"} }

    embed := &discordgo.MessageEmbed{
        Title:       fmt.Sprintf("💳 Guild Treasury: %s", guild.GuildName),
        Description: fmt.Sprintf("Balance: **%d GC**\nTotal Deposits: %d GC\nTotal Spent: %d GC\nMembers: %d", guild.Balance, guild.TotalDeposits, guild.TotalSpent, guild.MemberCount),
        Color:       0xf5b342,
        Fields: []*discordgo.MessageEmbedField{
            {Name: "Top Contributors", Value: strings.Join(topLines, "\n"), Inline: false},
        },
        Footer: &discordgo.MessageEmbedFooter{Text: "Deposits are permanent and fund Titan-tier servers."},
    }
    ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
    return nil
}

// ----------------------------------------------------------------------------
// Guild Join Command (/guild join <guild_id>) - placeholder (requires invite)
// ----------------------------------------------------------------------------

type GuildJoinCommand struct{}

func NewGuildJoinCommand() *GuildJoinCommand { return &GuildJoinCommand{} }
func (c *GuildJoinCommand) Name() string        { return "guild-join" }
func (c *GuildJoinCommand) Description() string { return "Request to join a guild (ask owner to invite)" }
func (c *GuildJoinCommand) RequiredPermission() bot.Permission { return bot.PermissionUser }

func (c *GuildJoinCommand) Execute(ctx *CommandContext) error {
    if len(ctx.Args) < 1 { return fmt.Errorf("usage: guild-join <guild_id>") }
    guildID := ctx.Args[0]
    // Placeholder: real invite workflow needed (tracking pending invites)
    embed := &discordgo.MessageEmbed{
        Title:       "📩 Guild Join Request",
        Description: fmt.Sprintf("To join `%s`, ask the owner/admin to run `guild-invite @you %s`." , guildID, guildID),
        Color:       0x9966ff,
    }
    ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
    return nil
}
