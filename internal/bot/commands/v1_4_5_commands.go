package commands

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// ============================================================================
// v1.4.0 COMMANDS - Medium Priority
// ============================================================================

// GiftCreditsCommand transfers credits between users
type GiftCreditsCommand struct{}

func (c *GiftCreditsCommand) Name() string { return "gift" }
func (c *GiftCreditsCommand) Description() string { return "Gift credits to another user" }
func (c *GiftCreditsCommand) RequiredPermission() bot.Permission { return bot.PermissionUser }

func (c *GiftCreditsCommand) Execute(ctx *CommandContext) error {
	if len(ctx.Args) < 2 || len(ctx.Message.Mentions) == 0 {
		return fmt.Errorf("usage: gift @user <amount>")
	}

	recipient := ctx.Message.Mentions[0]
	if recipient.ID == ctx.Message.Author.ID {
		return fmt.Errorf("cannot gift credits to yourself")
	}

	var amount int
	if _, err := fmt.Sscanf(ctx.Args[1], "%d", &amount); err != nil || amount <= 0 {
		return fmt.Errorf("amount must be a positive number")
	}

	if amount > 1000 {
		return fmt.Errorf("cannot gift more than 1000 credits at once")
	}

	// Get sender balance
	sender, err := ctx.DB.GetOrCreateUser(ctx.Message.Author.ID)
	if err != nil {
		return fmt.Errorf("failed to get your account: %v", err)
	}

	if sender.Credits < amount {
		return fmt.Errorf("insufficient credits. You have %d, trying to gift %d", sender.Credits, amount)
	}

	// Get recipient
	_, err = ctx.DB.GetOrCreateUser(recipient.ID)
	if err != nil {
		return fmt.Errorf("failed to get recipient account: %v", err)
	}

	// Perform transfer
	if err := ctx.DB.DeductCredits(ctx.Message.Author.ID, amount); err != nil {
		return fmt.Errorf("failed to deduct credits: %v", err)
	}

	if err := ctx.DB.AddCredits(recipient.ID, amount); err != nil {
		// Rollback
		ctx.DB.AddCredits(ctx.Message.Author.ID, amount)
		return fmt.Errorf("failed to add credits to recipient: %v", err)
	}

	// Log transaction
	ctx.DB.DB().Exec(`
		INSERT INTO credit_transactions (from_user, to_user, amount, transaction_type, description)
		VALUES ($1, $2, $3, $4, $5)
	`, ctx.Message.Author.ID, recipient.ID, amount, "gift", fmt.Sprintf("Gift from %s", ctx.Message.Author.Username))

	return ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, fmt.Sprintf(
		"💝 **Credit Gift Successful!**\n"+
			"From: %s\n"+
			"To: %s\n"+
			"Amount: %d credits\n"+
			"Your new balance: %d credits",
		ctx.Message.Author.Username, recipient.Username, amount, sender.Credits-amount,
	))
}

// TransactionsCommand shows credit transaction history
type TransactionsCommand struct{}

func (c *TransactionsCommand) Name() string { return "transactions" }
func (c *TransactionsCommand) Description() string { return "View credit transaction history" }
func (c *TransactionsCommand) RequiredPermission() bot.Permission { return bot.PermissionUser }

func (c *TransactionsCommand) Execute(ctx *CommandContext) error {
	rows, err := ctx.DB.DB().Query(`
		SELECT from_user, to_user, amount, transaction_type, description, created_at
		FROM credit_transactions
		WHERE from_user = $1 OR to_user = $1
		ORDER BY created_at DESC
		LIMIT 10
	`, ctx.Message.Author.ID)
	if err != nil {
		return fmt.Errorf("failed to fetch transactions: %v", err)
	}
	defer rows.Close()

	var history strings.Builder
	history.WriteString("💳 **Transaction History** (last 10)\n━━━━━━━━━━━━━━━━━━━━\n")

	count := 0
	for rows.Next() {
		var fromUser, toUser sql.NullString
		var amount int
		var txType, desc string
		var createdAt time.Time

		if err := rows.Scan(&fromUser, &toUser, &amount, &txType, &desc, &createdAt); err != nil {
			continue
		}

		direction := "+"
		if fromUser.Valid && fromUser.String == ctx.Message.Author.ID {
			direction = "-"
		}

		history.WriteString(fmt.Sprintf("%s%d credits • %s • %s\n", 
			direction, amount, txType, createdAt.Format("Jan 02")))
		count++
	}

	if count == 0 {
		history.WriteString("No transactions yet")
	}

	return ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, history.String())
}

// FavoriteCommand manages server favorites
type FavoriteCommand struct{}

func (c *FavoriteCommand) Name() string { return "favorite" }
func (c *FavoriteCommand) Description() string { return "Bookmark a server (add/remove/list)" }
func (c *FavoriteCommand) RequiredPermission() bot.Permission { return bot.PermissionUser }

func (c *FavoriteCommand) Execute(ctx *CommandContext) error {
	if len(ctx.Args) == 0 {
		return c.listFavorites(ctx)
	}

	action := strings.ToLower(ctx.Args[0])
	switch action {
	case "add":
		if len(ctx.Args) < 2 {
			return fmt.Errorf("usage: favorite add <server-id>")
		}
		return c.addFavorite(ctx, ctx.Args[1])
	case "remove", "rm":
		if len(ctx.Args) < 2 {
			return fmt.Errorf("usage: favorite remove <server-id>")
		}
		return c.removeFavorite(ctx, ctx.Args[1])
	case "list":
		return c.listFavorites(ctx)
	default:
		return fmt.Errorf("usage: favorite [add|remove|list] <server-id>")
	}
}

func (c *FavoriteCommand) addFavorite(ctx *CommandContext, serverIDStr string) error {
	var serverID int
	fmt.Sscanf(serverIDStr, "%d", &serverID)

	_, err := ctx.DB.DB().Exec(`
		INSERT INTO favorites (discord_id, server_id)
		VALUES ($1, $2)
		ON CONFLICT (discord_id, server_id) DO NOTHING
	`, ctx.Message.Author.ID, serverID)

	if err != nil {
		return fmt.Errorf("failed to add favorite: %v", err)
	}

	return ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, 
		fmt.Sprintf("⭐ Added server #%d to your favorites", serverID))
}

func (c *FavoriteCommand) removeFavorite(ctx *CommandContext, serverIDStr string) error {
	var serverID int
	fmt.Sscanf(serverIDStr, "%d", &serverID)

	_, err := ctx.DB.DB().Exec(`
		DELETE FROM favorites WHERE discord_id = $1 AND server_id = $2
	`, ctx.Message.Author.ID, serverID)

	if err != nil {
		return fmt.Errorf("failed to remove favorite: %v", err)
	}

	return ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, 
		fmt.Sprintf("🗑️ Removed server #%d from favorites", serverID))
}

func (c *FavoriteCommand) listFavorites(ctx *CommandContext) error {
	rows, err := ctx.DB.DB().Query(`
		SELECT f.server_id, ps.server_name, ps.game_type, ps.owner_name
		FROM favorites f
		JOIN public_servers ps ON f.server_id = ps.id
		WHERE f.discord_id = $1
		ORDER BY f.added_at DESC
	`, ctx.Message.Author.ID)
	if err != nil {
		return fmt.Errorf("failed to fetch favorites: %v", err)
	}
	defer rows.Close()

	var favorites strings.Builder
	favorites.WriteString("⭐ **Your Favorite Servers**\n━━━━━━━━━━━━━━━━━━━━\n")

	count := 0
	for rows.Next() {
		var id int
		var name, game, owner string
		if err := rows.Scan(&id, &name, &game, &owner); err != nil {
			continue
		}
		favorites.WriteString(fmt.Sprintf("**#%d** %s (%s) - by %s\n", id, name, game, owner))
		count++
	}

	if count == 0 {
		favorites.WriteString("No favorites yet. Use `favorite add <server-id>` to bookmark servers")
	}

	return ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, favorites.String())
}

// SearchServersCommand searches public lobby
type SearchServersCommand struct{}

func (c *SearchServersCommand) Name() string { return "search" }
func (c *SearchServersCommand) Description() string { return "Search public servers" }
func (c *SearchServersCommand) RequiredPermission() bot.Permission { return bot.PermissionUser }

func (c *SearchServersCommand) Execute(ctx *CommandContext) error {
	if len(ctx.Args) == 0 {
		return fmt.Errorf("usage: search <game-type or keyword>")
	}

	query := strings.ToLower(strings.Join(ctx.Args, " "))

	rows, err := ctx.DB.DB().Query(`
		SELECT id, server_name, game_type, owner_name, players, max_players, description
		FROM public_servers
		WHERE LOWER(game_type) LIKE $1 OR LOWER(server_name) LIKE $1 OR LOWER(description) LIKE $1
		ORDER BY players DESC
		LIMIT 10
	`, "%"+query+"%")
	if err != nil {
		return fmt.Errorf("search failed: %v", err)
	}
	defer rows.Close()

	var results strings.Builder
	results.WriteString(fmt.Sprintf("🔍 **Search Results for '%s'**\n━━━━━━━━━━━━━━━━━━━━\n", query))

	count := 0
	for rows.Next() {
		var id, players, maxPlayers int
		var name, game, owner, desc string
		if err := rows.Scan(&id, &name, &game, &owner, &players, &maxPlayers, &desc); err != nil {
			continue
		}

		results.WriteString(fmt.Sprintf("**#%d** %s (%s)\n", id, name, game))
		results.WriteString(fmt.Sprintf("  👤 %d/%d players • by %s\n", players, maxPlayers, owner))
		if desc != "" {
			results.WriteString(fmt.Sprintf("  📝 %s\n", desc))
		}
		results.WriteString("\n")
		count++
	}

	if count == 0 {
		results.WriteString("No servers found matching your query")
	}

	return ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, results.String())
}

// ShopCommand shows purchasable items
type ShopCommand struct{}

func (c *ShopCommand) Name() string { return "shop" }
func (c *ShopCommand) Description() string { return "Browse shop items" }
func (c *ShopCommand) RequiredPermission() bot.Permission { return bot.PermissionUser }

func (c *ShopCommand) Execute(ctx *CommandContext) error {
	rows, err := ctx.DB.DB().Query(`
		SELECT id, item_name, item_type, description, price
		FROM shop_items
		WHERE is_active = true
		ORDER BY price ASC
	`)
	if err != nil {
		return fmt.Errorf("failed to load shop: %v", err)
	}
	defer rows.Close()

	var shop strings.Builder
	shop.WriteString("🛒 **AGIS Shop**\n━━━━━━━━━━━━━━━━━━━━\n")

	count := 0
	for rows.Next() {
		var id, price int
		var name, itemType, desc string
		if err := rows.Scan(&id, &name, &itemType, &desc, &price); err != nil {
			continue
		}

		shop.WriteString(fmt.Sprintf("**[%d]** %s - %d credits\n", id, name, price))
		shop.WriteString(fmt.Sprintf("  Type: %s\n", itemType))
		if desc != "" {
			shop.WriteString(fmt.Sprintf("  %s\n", desc))
		}
		shop.WriteString("\n")
		count++
	}

	if count == 0 {
		shop.WriteString("Shop is empty. Check back later!")
	} else {
		shop.WriteString("Use `buy <item-id>` to purchase")
	}

	return ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, shop.String())
}

// ============================================================================
// v1.5.0 COMMANDS - Low Priority
// ============================================================================

// AchievementsCommand shows user achievements
type AchievementsCommand struct{}

func (c *AchievementsCommand) Name() string { return "achievements" }
func (c *AchievementsCommand) Description() string { return "View your achievements" }
func (c *AchievementsCommand) RequiredPermission() bot.Permission { return bot.PermissionUser }

func (c *AchievementsCommand) Execute(ctx *CommandContext) error {
	// Get unlocked achievements
	rows, err := ctx.DB.DB().Query(`
		SELECT a.icon, a.name, a.description, ua.unlocked_at
		FROM user_achievements ua
		JOIN achievements a ON ua.achievement_id = a.id
		WHERE ua.discord_id = $1
		ORDER BY ua.unlocked_at DESC
	`, ctx.Message.Author.ID)
	if err != nil {
		return fmt.Errorf("failed to load achievements: %v", err)
	}
	defer rows.Close()

	var achievements strings.Builder
	achievements.WriteString("🏆 **Your Achievements**\n━━━━━━━━━━━━━━━━━━━━\n")

	count := 0
	for rows.Next() {
		var icon, name, desc string
		var unlockedAt time.Time
		if err := rows.Scan(&icon, &name, &desc, &unlockedAt); err != nil {
			continue
		}

		achievements.WriteString(fmt.Sprintf("%s **%s**\n", icon, name))
		achievements.WriteString(fmt.Sprintf("  %s\n", desc))
		achievements.WriteString(fmt.Sprintf("  Unlocked: %s\n\n", unlockedAt.Format("Jan 02, 2006")))
		count++
	}

	if count == 0 {
		achievements.WriteString("No achievements unlocked yet.\nKeep playing to earn rewards!")
	}

	// Show total count
	var total int
	ctx.DB.DB().QueryRow(`SELECT COUNT(*) FROM achievements`).Scan(&total)
	achievements.WriteString(fmt.Sprintf("\nProgress: %d/%d achievements unlocked", count, total))

	return ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, achievements.String())
}

// ReviewCommand manages server reviews
type ReviewCommand struct{}

func (c *ReviewCommand) Name() string { return "review" }
func (c *ReviewCommand) Description() string { return "Review a public server (1-5 stars)" }
func (c *ReviewCommand) RequiredPermission() bot.Permission { return bot.PermissionUser }

func (c *ReviewCommand) Execute(ctx *CommandContext) error {
	if len(ctx.Args) < 3 {
		return fmt.Errorf("usage: review <server-id> <rating 1-5> <comment>")
	}

	var serverID, rating int
	fmt.Sscanf(ctx.Args[0], "%d", &serverID)
	fmt.Sscanf(ctx.Args[1], "%d", &rating)

	if rating < 1 || rating > 5 {
		return fmt.Errorf("rating must be between 1 and 5")
	}

	comment := strings.Join(ctx.Args[2:], " ")
	if len(comment) > 500 {
		return fmt.Errorf("comment too long (max 500 characters)")
	}

	_, err := ctx.DB.DB().Exec(`
		INSERT INTO server_reviews (server_id, reviewer_id, rating, comment)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (server_id, reviewer_id) 
		DO UPDATE SET rating = $3, comment = $4, created_at = CURRENT_TIMESTAMP
	`, serverID, ctx.Message.Author.ID, rating, comment)

	if err != nil {
		return fmt.Errorf("failed to submit review: %v", err)
	}

	stars := strings.Repeat("⭐", rating)
	return ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, fmt.Sprintf(
		"✅ **Review Submitted!**\n"+
			"Server: #%d\n"+
			"Rating: %s (%d/5)\n"+
			"Comment: %s",
		serverID, stars, rating, comment,
	))
}

// ReviewsCommand shows server reviews
type ReviewsCommand struct{}

func (c *ReviewsCommand) Name() string { return "reviews" }
func (c *ReviewsCommand) Description() string { return "View server reviews" }
func (c *ReviewsCommand) RequiredPermission() bot.Permission { return bot.PermissionUser }

func (c *ReviewsCommand) Execute(ctx *CommandContext) error {
	if len(ctx.Args) == 0 {
		return fmt.Errorf("usage: reviews <server-id>")
	}

	var serverID int
	fmt.Sscanf(ctx.Args[0], "%d", &serverID)

	// Get average rating
	var avgRating float64
	var reviewCount int
	ctx.DB.DB().QueryRow(`
		SELECT COALESCE(AVG(rating), 0), COUNT(*)
		FROM server_reviews
		WHERE server_id = $1
	`, serverID).Scan(&avgRating, &reviewCount)

	rows, err := ctx.DB.DB().Query(`
		SELECT rating, comment, created_at
		FROM server_reviews
		WHERE server_id = $1
		ORDER BY created_at DESC
		LIMIT 5
	`, serverID)
	if err != nil {
		return fmt.Errorf("failed to load reviews: %v", err)
	}
	defer rows.Close()

	var reviews strings.Builder
	reviews.WriteString(fmt.Sprintf("📝 **Reviews for Server #%d**\n", serverID))
	reviews.WriteString("━━━━━━━━━━━━━━━━━━━━\n")
	reviews.WriteString(fmt.Sprintf("Average: %.1f/5 ⭐ (%d reviews)\n\n", avgRating, reviewCount))

	for rows.Next() {
		var rating int
		var comment string
		var createdAt time.Time
		if err := rows.Scan(&rating, &comment, &createdAt); err != nil {
			continue
		}

		stars := strings.Repeat("⭐", rating)
		reviews.WriteString(fmt.Sprintf("%s %s\n", stars, createdAt.Format("Jan 02")))
		reviews.WriteString(fmt.Sprintf("  \"%s\"\n\n", comment))
	}

	return ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, reviews.String())
}
