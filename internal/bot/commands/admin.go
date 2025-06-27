package commands

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"agis-bot/internal/bot"
)

type AdminStatusCommand struct{}

func (c *AdminStatusCommand) Name() string {
	return "admin"
}

func (c *AdminStatusCommand) Description() string {
	return "Admin cluster management commands"
}

func (c *AdminStatusCommand) RequiredPermission() bot.Permission {
	return bot.PermissionAdmin
}

func (c *AdminStatusCommand) Execute(ctx *CommandContext) error {
	// Check subcommand
	if len(ctx.Args) == 0 {
		return c.showAdminHelp(ctx)
	}

	subcommand := ctx.Args[0]
	switch subcommand {
	case "status":
		return c.handleStatus(ctx)
	case "pods":
		return c.handlePods(ctx)
	case "nodes":
		return c.handleNodes(ctx)
	case "credits":
		return c.handleCredits(ctx)
	default:
		return c.showAdminHelp(ctx)
	}
}

func (c *AdminStatusCommand) showAdminHelp(ctx *CommandContext) error {
	embed := &discordgo.MessageEmbed{
		Title: "⚙️ Admin Commands",
		Color: 0xff6600,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Cluster Management",
				Value:  "`admin status` - Cluster health status\n`admin pods` - List all pods\n`admin nodes` - List cluster nodes",
				Inline: false,
			},
			{
				Name:   "Credit Management",
				Value:  "`admin credits add @user <amount>` - Add credits to user\n`admin credits remove @user <amount>` - Remove credits from user\n`admin credits check @user` - Check user's credits",
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "🔒 Admin-only commands • Use with caution",
		},
	}
	_, err := ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	return err
}

func (c *AdminStatusCommand) handleStatus(ctx *CommandContext) error {
	embed := &discordgo.MessageEmbed{
		Title: "🎯 WTG Cluster Status",
		Color: 0x00ff00,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "📊 Cluster Health",
				Value:  "✅ Operational",
				Inline: true,
			},
			{
				Name:   "🤖 Bot Status",
				Value:  "✅ Online",
				Inline: true,
			},
			{
				Name:   "🔐 Permission Level",
				Value:  fmt.Sprintf("✅ %s Access", bot.GetPermissionString(ctx.UserPerm)),
				Inline: true,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Agis - WTG Cluster Management Bot (Admin Mode)",
		},
	}

	_, err := ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	return err
}

func (c *AdminStatusCommand) handlePods(ctx *CommandContext) error {
	// Get Kubernetes client
	config, err := rest.InClusterConfig()
	if err != nil {
		return fmt.Errorf("failed to get Kubernetes config: %v", err)
	}

	k8sClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %v", err)
	}

	// Get pods from wtg-dev namespace
	pods, err := k8sClient.CoreV1().Pods("wtg-dev").List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to get pods: %v", err)
	}

	var fields []*discordgo.MessageEmbedField
	for _, pod := range pods.Items {
		var status string
		switch pod.Status.Phase {
		case "Running":
			status = "✅ Running"
		case "Pending":
			status = "⏳ Pending"
		default:
			status = "❌ " + string(pod.Status.Phase)
		}

		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   pod.Name,
			Value:  status,
			Inline: true,
		})
	}

	embed := &discordgo.MessageEmbed{
		Title:  "🚢 WTG-Dev Pods",
		Color:  0x0099ff,
		Fields: fields,
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Total: %d pods • Admin access required", len(pods.Items)),
		},
	}

	_, err = ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	return err
}

func (c *AdminStatusCommand) handleNodes(ctx *CommandContext) error {
	// Get Kubernetes client
	config, err := rest.InClusterConfig()
	if err != nil {
		return fmt.Errorf("failed to get Kubernetes config: %v", err)
	}

	k8sClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %v", err)
	}

	nodes, err := k8sClient.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to get nodes: %v", err)
	}

	var fields []*discordgo.MessageEmbedField
	for _, node := range nodes.Items {
		ready := "❌ Not Ready"
		for _, condition := range node.Status.Conditions {
			if condition.Type == "Ready" && condition.Status == "True" {
				ready = "✅ Ready"
				break
			}
		}

		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   node.Name,
			Value:  ready,
			Inline: true,
		})
	}

	embed := &discordgo.MessageEmbed{
		Title:  "🖥️ Cluster Nodes",
		Color:  0x9932cc,
		Fields: fields,
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Total: %d nodes • Admin access required", len(nodes.Items)),
		},
	}

	_, err = ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	return err
}

func (c *AdminStatusCommand) handleCredits(ctx *CommandContext) error {
	if len(ctx.Args) < 2 {
		embed := &discordgo.MessageEmbed{
			Title: "💰 Admin Credit Management",
			Color: 0xff6600,
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:  "Available Commands",
					Value: "`admin credits add @user <amount>` - Add credits to a user\n`admin credits remove @user <amount>` - Remove credits from a user\n`admin credits check @user` - Check user's credits",
				},
				{
					Name:  "Examples",
					Value: "`admin credits add @username 100`\n`admin credits remove @username 50`\n`admin credits check @username`",
				},
			},
			Footer: &discordgo.MessageEmbedFooter{
				Text: "🔒 Admin-only • Changes are immediate",
			},
		}
		_, err := ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
		return err
	}

	action := ctx.Args[1]
	switch action {
	case "add":
		return c.handleAddCredits(ctx)
	case "remove":
		return c.handleRemoveCredits(ctx)
	case "check":
		return c.handleCheckCredits(ctx)
	default:
		embed := &discordgo.MessageEmbed{
			Title:       "❌ Invalid Action",
			Description: fmt.Sprintf("Action '%s' is not recognized", action),
			Color:       0xff0000,
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:  "Valid Actions",
					Value: "add, remove, check",
				},
			},
		}
		_, err := ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
		return err
	}
}

func (c *AdminStatusCommand) handleAddCredits(ctx *CommandContext) error {
	if len(ctx.Args) < 4 || !strings.HasPrefix(ctx.Args[2], "<@") {
		embed := &discordgo.MessageEmbed{
			Title:       "❌ Invalid Usage",
			Description: "Mention a user and specify the amount to add",
			Color:       0xff0000,
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:  "Usage",
					Value: "`admin credits add @user <amount>`",
				},
				{
					Name:  "Example",
					Value: "`admin credits add @username 100`",
				},
			},
		}
		_, err := ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
		return err
	}

	// Extract user ID from mention
	userMention := ctx.Args[2]
	userID := strings.Trim(userMention, "<@!>")

	// Parse amount
	amount, err := strconv.Atoi(ctx.Args[3])
	if err != nil || amount <= 0 {
		embed := &discordgo.MessageEmbed{
			Title:       "❌ Invalid Amount",
			Description: "Amount must be a positive number",
			Color:       0xff0000,
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:  "Example",
					Value: "`admin credits add @username 100`",
				},
			},
		}
		_, err := ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
		return err
	}

	// Get or create the target user
	targetUser, err := ctx.DB.GetOrCreateUser(userID)
	if err != nil {
		return fmt.Errorf("failed to get target user: %v", err)
	}

	oldBalance := targetUser.Credits

	// Add credits
	err = ctx.DB.AddCredits(userID, amount)
	if err != nil {
		return fmt.Errorf("failed to add credits: %v", err)
	}

	// Get updated balance
	updatedUser, err := ctx.DB.GetOrCreateUser(userID)
	if err != nil {
		return fmt.Errorf("failed to get updated user: %v", err)
	}

	embed := &discordgo.MessageEmbed{
		Title:       "✅ Credits Added Successfully",
		Description: fmt.Sprintf("Added %d credits to <@%s>", amount, userID),
		Color:       0x00ff00,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Transaction Details",
				Value:  fmt.Sprintf("**Amount Added:** %d credits\n**Previous Balance:** %d credits\n**New Balance:** %d credits", amount, oldBalance, updatedUser.Credits),
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Admin action by %s", ctx.Message.Author.Username),
		},
	}

	_, err = ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	return err
}

func (c *AdminStatusCommand) handleRemoveCredits(ctx *CommandContext) error {
	if len(ctx.Args) < 4 || !strings.HasPrefix(ctx.Args[2], "<@") {
		embed := &discordgo.MessageEmbed{
			Title:       "❌ Invalid Usage",
			Description: "Mention a user and specify the amount to remove",
			Color:       0xff0000,
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:  "Usage",
					Value: "`admin credits remove @user <amount>`",
				},
				{
					Name:  "Example",
					Value: "`admin credits remove @username 50`",
				},
			},
		}
		_, err := ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
		return err
	}

	// Extract user ID from mention
	userMention := ctx.Args[2]
	userID := strings.Trim(userMention, "<@!>")

	// Parse amount
	amount, err := strconv.Atoi(ctx.Args[3])
	if err != nil || amount <= 0 {
		embed := &discordgo.MessageEmbed{
			Title:       "❌ Invalid Amount",
			Description: "Amount must be a positive number",
			Color:       0xff0000,
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:  "Example",
					Value: "`admin credits remove @username 50`",
				},
			},
		}
		_, err := ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
		return err
	}

	// Get or create the target user
	targetUser, err := ctx.DB.GetOrCreateUser(userID)
	if err != nil {
		return fmt.Errorf("failed to get target user: %v", err)
	}

	oldBalance := targetUser.Credits

	// Remove credits (add negative amount)
	err = ctx.DB.AddCredits(userID, -amount)
	if err != nil {
		return fmt.Errorf("failed to remove credits: %v", err)
	}

	// Get updated balance
	updatedUser, err := ctx.DB.GetOrCreateUser(userID)
	if err != nil {
		return fmt.Errorf("failed to get updated user: %v", err)
	}

	embed := &discordgo.MessageEmbed{
		Title:       "✅ Credits Removed Successfully",
		Description: fmt.Sprintf("Removed %d credits from <@%s>", amount, userID),
		Color:       0xff9900,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Transaction Details",
				Value:  fmt.Sprintf("**Amount Removed:** %d credits\n**Previous Balance:** %d credits\n**New Balance:** %d credits", amount, oldBalance, updatedUser.Credits),
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Admin action by %s", ctx.Message.Author.Username),
		},
	}

	_, err = ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	return err
}

func (c *AdminStatusCommand) handleCheckCredits(ctx *CommandContext) error {
	if len(ctx.Args) < 3 || !strings.HasPrefix(ctx.Args[2], "<@") {
		embed := &discordgo.MessageEmbed{
			Title:       "❌ Invalid Usage",
			Description: "Mention a user to check their credits",
			Color:       0xff0000,
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:  "Usage",
					Value: "`admin credits check @user`",
				},
				{
					Name:  "Example",
					Value: "`admin credits check @username`",
				},
			},
		}
		_, err := ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
		return err
	}

	// Extract user ID from mention
	userMention := ctx.Args[2]
	userID := strings.Trim(userMention, "<@!>")

	// Get or create the target user
	targetUser, err := ctx.DB.GetOrCreateUser(userID)
	if err != nil {
		return fmt.Errorf("failed to get target user: %v", err)
	}

	embed := &discordgo.MessageEmbed{
		Title:       "💰 User Credit Balance",
		Description: fmt.Sprintf("Credit information for <@%s>", userID),
		Color:       0x0099ff,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Current Balance",
				Value:  fmt.Sprintf("**%d credits**", targetUser.Credits),
				Inline: true,
			},
			{
				Name:   "User ID",
				Value:  userID,
				Inline: true,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Admin query by %s", ctx.Message.Author.Username),
		},
	}

	_, err = ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	return err
}
