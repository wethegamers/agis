package commands

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"agis-bot/internal/bot"
)

// ============================================================================
// SUBSCRIPTION TIER SYSTEM (Economy Plan v2.0)
// ============================================================================

// SubscribeCommand - Manage premium subscription
type SubscribeCommand struct{}

func (c *SubscribeCommand) Name() string                             { return "subscribe" }
func (c *SubscribeCommand) Description() string                      { return "Manage your premium subscription" }
func (c *SubscribeCommand) RequiredPermission() bot.Permission { return bot.PermissionUser }

func (c *SubscribeCommand) Execute(ctx *CommandContext) error {
	if len(ctx.Args) == 0 {
		return c.showSubscriptionInfo(ctx)
	}

	action := strings.ToLower(ctx.Args[0])

	switch action {
	case "activate", "start":
		return c.activateSubscription(ctx)
	case "cancel", "stop":
		return c.cancelSubscription(ctx)
	case "status":
		return c.showSubscriptionStatus(ctx)
	default:
		return fmt.Errorf("unknown action. Use: subscribe [activate|cancel|status]")
	}
}

func (c *SubscribeCommand) showSubscriptionInfo(ctx *CommandContext) error {
	// Check if user already has subscription
	var tier string
	var expiresAt sql.NullTime
	err := ctx.DB.DB().QueryRow(`
		SELECT tier, subscription_expires 
		FROM users 
		WHERE discord_id = $1
	`, ctx.Message.Author.ID).Scan(&tier, &expiresAt)

	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("failed to check subscription: %v", err)
	}

	var output strings.Builder
	output.WriteString("💎 **WeTheGamers Premium Subscription**\n")
	output.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

	if tier == "premium" && expiresAt.Valid && expiresAt.Time.After(time.Now()) {
		output.WriteString("✅ **You are subscribed!**\n\n")
		output.WriteString(fmt.Sprintf("📅 Expires: %s\n", expiresAt.Time.Format("2006-01-02 15:04")))
		output.WriteString(fmt.Sprintf("⏱️ Days remaining: %d\n\n", int(time.Until(expiresAt.Time).Hours()/24)))
		output.WriteString("Use `subscribe cancel` to cancel your subscription\n")
	} else {
		output.WriteString("**Premium Benefits - $3.99/month**\n\n")
		output.WriteString("🎁 **5 WTG Allowance** - $5.00 value monthly\n")
		output.WriteString("🆓 **Free Server Rent** - 3000 GC/month waived\n")
		output.WriteString("⚡ **2x GC Multiplier** - Earn double from ads & work\n")
		output.WriteString("🎯 **Enhanced Daily Bonus** - 100 GC instead of 50\n")
		output.WriteString("👑 **Exclusive Premium Role** - Stand out in the community\n")
		output.WriteString("🚀 **Priority Support** - Faster response times\n")
		output.WriteString("📊 **Advanced Stats** - Detailed analytics\n\n")
		output.WriteString("💡 **Value Proposition:**\n")
		output.WriteString("Pay $3.99, get $5.00 worth of WTG + free 3000 GC server!\n")
		output.WriteString("That's an instant profit even before the multipliers!\n\n")
		output.WriteString("⚠️ **Payment Integration Coming Soon**\n")
		output.WriteString("Admins can activate with: `subscribe activate @user`\n")
	}

	_, err = ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, output.String())
	return err
}

func (c *SubscribeCommand) activateSubscription(ctx *CommandContext) error {
	// TODO: Integrate with Stripe/PayPal for actual payment processing
	// For now, this is admin-only manual activation

	if !ctx.Permissions.IsAdmin(ctx.Session, ctx.Message.GuildID, ctx.Message.Author.ID) {
		return fmt.Errorf("❌ Payment integration coming soon! Admins can manually activate subscriptions for testing.")
	}

	// Admin activating for another user
	if len(ctx.Args) < 2 {
		return fmt.Errorf("usage: subscribe activate @user [days]")
	}

	// Extract user mention
	userID := ctx.Args[1]
	userID = strings.Trim(userID, "<@!>")

	days := 30 // Default 1 month
	if len(ctx.Args) > 2 {
		fmt.Sscanf(ctx.Args[2], "%d", &days)
	}

	// Activate subscription
	expiresAt := time.Now().Add(time.Duration(days) * 24 * time.Hour)

	tx, err := ctx.DB.DB().Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %v", err)
	}
	defer tx.Rollback()

	// Update user tier and expiration
	_, err = tx.Exec(`
		UPDATE users 
		SET tier = 'premium', 
		    subscription_expires = $1,
		    wtg_coins = wtg_coins + 5
		WHERE discord_id = $2
	`, expiresAt, userID)

	if err != nil {
		return fmt.Errorf("failed to activate subscription: %v", err)
	}

	// Log transaction
	_, err = tx.Exec(`
		INSERT INTO credit_transactions (from_user, to_user, amount, transaction_type, description, currency_type)
		VALUES ('SYSTEM', $1, 5, 'subscription', 'Premium subscription activated - 5 WTG allowance', 'WTG')
	`, userID)

	if err != nil {
		return fmt.Errorf("failed to log transaction: %v", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit subscription: %v", err)
	}

	_, err = ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, fmt.Sprintf(
		"✅ **Premium Subscription Activated!**\n\n"+
			"User: <@%s>\n"+
			"Duration: %d days\n"+
			"Expires: %s\n"+
			"WTG Granted: 5\n\n"+
			"Benefits now active!",
		userID, days, expiresAt.Format("2006-01-02 15:04")))
	return err
}

func (c *SubscribeCommand) cancelSubscription(ctx *CommandContext) error {
	// Cancel subscription
	var tier string
	var expiresAt sql.NullTime
	err := ctx.DB.DB().QueryRow(`
		SELECT tier, subscription_expires 
		FROM users 
		WHERE discord_id = $1
	`, ctx.Message.Author.ID).Scan(&tier, &expiresAt)

	if err == sql.ErrNoRows || tier != "premium" {
		return fmt.Errorf("you don't have an active subscription to cancel")
	}

	if err != nil {
		return fmt.Errorf("failed to check subscription: %v", err)
	}

	// Set tier back to free but maintain expiration date (benefits until end of period)
	_, err = ctx.DB.DB().Exec(`
		UPDATE users 
		SET tier = 'free' 
		WHERE discord_id = $1
	`, ctx.Message.Author.ID)

	if err != nil {
		return fmt.Errorf("failed to cancel subscription: %v", err)
	}

	_, err = ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, fmt.Sprintf(
		"✅ **Subscription Cancelled**\n\n"+
			"Your premium benefits will remain active until: %s\n"+
			"After this date, you'll return to the free tier.\n\n"+
			"We're sorry to see you go! Use `subscribe` anytime to reactivate.",
		expiresAt.Time.Format("2006-01-02 15:04")))
	return err
}

func (c *SubscribeCommand) showSubscriptionStatus(ctx *CommandContext) error {
	var tier string
	var expiresAt sql.NullTime
	var wtgCoins, credits int

	err := ctx.DB.DB().QueryRow(`
		SELECT tier, subscription_expires, COALESCE(wtg_coins, 0), credits
		FROM users 
		WHERE discord_id = $1
	`, ctx.Message.Author.ID).Scan(&tier, &expiresAt, &wtgCoins, &credits)

	if err == sql.ErrNoRows {
		return fmt.Errorf("user not found")
	}

	if err != nil {
		return fmt.Errorf("failed to get subscription status: %v", err)
	}

	var output strings.Builder
	output.WriteString("📊 **Your Subscription Status**\n")
	output.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

	if tier == "premium" && expiresAt.Valid && expiresAt.Time.After(time.Now()) {
		output.WriteString("**Tier:** 👑 Premium\n")
		output.WriteString(fmt.Sprintf("**Expires:** %s\n", expiresAt.Time.Format("2006-01-02 15:04")))
		daysLeft := int(time.Until(expiresAt.Time).Hours() / 24)
		output.WriteString(fmt.Sprintf("**Days Remaining:** %d\n\n", daysLeft))

		output.WriteString("**Active Benefits:**\n")
		output.WriteString("✅ 2x GC multiplier on ads & work\n")
		output.WriteString("✅ Enhanced daily bonus (100 GC)\n")
		output.WriteString("✅ Free 3000 GC server rent\n")
		output.WriteString("✅ Priority support\n")
		output.WriteString("✅ Premium role & badge\n\n")

		if daysLeft <= 7 {
			output.WriteString("⚠️ **Renewal Reminder**\n")
			output.WriteString("Your subscription expires soon! Renew to keep your benefits.\n")
		}
	} else {
		output.WriteString("**Tier:** Free\n")
		if expiresAt.Valid {
			output.WriteString(fmt.Sprintf("**Last Subscription:** %s\n\n", expiresAt.Time.Format("2006-01-02")))
		} else {
			output.WriteString("**Status:** Never subscribed\n\n")
		}

		output.WriteString("💎 **Upgrade to Premium?**\n")
		output.WriteString("Use `subscribe` to see premium benefits!\n")
	}

	output.WriteString(fmt.Sprintf("\n**Current Balances:**\n"))
	output.WriteString(fmt.Sprintf("💎 WTG Coins: %d\n", wtgCoins))
	output.WriteString(fmt.Sprintf("💰 GameCredits: %d\n", credits))

	_, err = ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, output.String())
	return err
}

// Helper function to check if user has active premium subscription
func HasActivePremium(db *sql.DB, discordID string) bool {
	var tier string
	var expiresAt sql.NullTime

	err := db.QueryRow(`
		SELECT tier, subscription_expires 
		FROM users 
		WHERE discord_id = $1
	`, discordID).Scan(&tier, &expiresAt)

	if err != nil {
		return false
	}

	return tier == "premium" && expiresAt.Valid && expiresAt.Time.After(time.Now())
}

// GetUserMultiplier returns the GC earning multiplier for a user (1x or 2x for premium)
func GetUserMultiplier(db *sql.DB, discordID string) int {
	if HasActivePremium(db, discordID) {
		return 2
	}
	return 1
}
