package commands

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"agis-bot/internal/bot"
	"agis-bot/internal/services"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// ============================================================================
// v1.6.0 Features:
// 1. Real Kubernetes log streaming
// 2. BotKube-style cluster query commands
// 3. Shop purchase system with WTG/GC support
// 4. WTG currency conversion
// ============================================================================

// ============================================================================
// REAL KUBERNETES LOG STREAMING
// ============================================================================

// K8sLogsCommand - replacement for placeholder logs command with real streaming
type K8sLogsCommand struct{}

func (c *K8sLogsCommand) Name() string                             { return "logs" }
func (c *K8sLogsCommand) Description() string                      { return "Stream server logs from Kubernetes" }
func (c *K8sLogsCommand) RequiredPermission() bot.Permission { return bot.PermissionUser }

func (c *K8sLogsCommand) Execute(ctx *CommandContext) error {
	if len(ctx.Args) == 0 {
		return fmt.Errorf("usage: logs <server-name> [lines]")
	}

	serverName := ctx.Args[0]
	lines := int64(50)
	if len(ctx.Args) > 1 {
		if parsed, err := strconv.ParseInt(ctx.Args[1], 10, 64); err == nil {
			lines = parsed
			if lines > 200 {
				lines = 200
			}
		}
	}

	// Get server from database
	servers, err := ctx.DB.GetUserServers(ctx.Message.Author.ID)
	if err != nil {
		return fmt.Errorf("failed to get servers: %v", err)
	}

	var targetServer *services.GameServer
	for _, srv := range servers {
		if srv.Name == serverName {
			targetServer = srv
			break
		}
	}

	if targetServer == nil {
		return fmt.Errorf("server '%s' not found", serverName)
	}

	// Create Kubernetes client
	k8sClient, err := getKubernetesClient()
	if err != nil {
		return fmt.Errorf("failed to connect to cluster: %v", err)
	}

	// Find pod for this server
	namespace := "game-servers" // Default namespace for game servers
	podList, err := k8sClient.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{
		LabelSelector: fmt.Sprintf("server-name=%s", serverName),
	})

	if err != nil || len(podList.Items) == 0 {
		return fmt.Errorf("server pod not found or not yet created")
	}

	pod := podList.Items[0]

	// Get logs
	logOptions := &corev1.PodLogOptions{
		TailLines: &lines,
	}

	req := k8sClient.CoreV1().Pods(namespace).GetLogs(pod.Name, logOptions)
	logStream, err := req.Stream(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get logs: %v", err)
	}
	defer logStream.Close()

	logBytes, err := io.ReadAll(logStream)
	if err != nil {
		return fmt.Errorf("failed to read logs: %v", err)
	}

	logContent := string(logBytes)
	if len(logContent) == 0 {
		logContent = "[No logs available yet]"
	}

	// Discord messages have 2000 char limit, split if needed
	maxLen := 1900 // Leave room for formatting
	if len(logContent) > maxLen {
		logContent = "..." + logContent[len(logContent)-maxLen:]
	}

	response := fmt.Sprintf("📋 **Logs for %s** (last %d lines)\n```\n%s\n```", 
		serverName, lines, logContent)

	return ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, response)
}

// ============================================================================
// BOTKUBE-STYLE CLUSTER COMMANDS (ClusterAdmin role required)
// ============================================================================

// ClusterPodsCommand - List pods across namespaces
type ClusterPodsCommand struct{}

func (c *ClusterPodsCommand) Name() string                             { return "cluster-pods" }
func (c *ClusterPodsCommand) Description() string                      { return "List pods in cluster" }
func (c *ClusterPodsCommand) RequiredPermission() bot.Permission { return bot.PermissionClusterAdmin }

func (c *ClusterPodsCommand) Execute(ctx *CommandContext) error {
	namespace := "game-servers"
	if len(ctx.Args) > 0 {
		namespace = ctx.Args[0]
	}

	k8sClient, err := getKubernetesClient()
	if err != nil {
		return fmt.Errorf("failed to connect to cluster: %v", err)
	}

	pods, err := k8sClient.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list pods: %v", err)
	}

	var output strings.Builder
	output.WriteString(fmt.Sprintf("🔷 **Pods in namespace `%s`** (%d total)\n", namespace, len(pods.Items)))
	output.WriteString("```\n")
	output.WriteString(fmt.Sprintf("%-40s %-15s %s\n", "NAME", "STATUS", "AGE"))
	output.WriteString(strings.Repeat("-", 80) + "\n")

	for _, pod := range pods.Items {
		age := time.Since(pod.CreationTimestamp.Time).Round(time.Second)
		status := string(pod.Status.Phase)
		
		// Truncate long names
		name := pod.Name
		if len(name) > 38 {
			name = name[:35] + "..."
		}
		
		output.WriteString(fmt.Sprintf("%-40s %-15s %s\n", name, status, formatDuration(age)))
	}
	output.WriteString("```")

	return ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, output.String())
}

// ClusterNodesCommand - List cluster nodes
type ClusterNodesCommand struct{}

func (c *ClusterNodesCommand) Name() string                             { return "cluster-nodes" }
func (c *ClusterNodesCommand) Description() string                      { return "List cluster nodes" }
func (c *ClusterNodesCommand) RequiredPermission() bot.Permission { return bot.PermissionClusterAdmin }

func (c *ClusterNodesCommand) Execute(ctx *CommandContext) error {
	k8sClient, err := getKubernetesClient()
	if err != nil {
		return fmt.Errorf("failed to connect to cluster: %v", err)
	}

	nodes, err := k8sClient.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list nodes: %v", err)
	}

	var output strings.Builder
	output.WriteString(fmt.Sprintf("🖥️ **Cluster Nodes** (%d total)\n", len(nodes.Items)))
	output.WriteString("```\n")

	for _, node := range nodes.Items {
		status := "NotReady"
		for _, cond := range node.Status.Conditions {
			if cond.Type == corev1.NodeReady && cond.Status == corev1.ConditionTrue {
				status = "Ready"
				break
			}
		}

		cpu := node.Status.Capacity[corev1.ResourceCPU]
		memory := node.Status.Capacity[corev1.ResourceMemory]
		age := time.Since(node.CreationTimestamp.Time).Round(time.Second)

		output.WriteString(fmt.Sprintf("%s\n", node.Name))
		output.WriteString(fmt.Sprintf("  Status: %s | CPU: %s | Memory: %s | Age: %s\n\n", 
			status, cpu.String(), memory.String(), formatDuration(age)))
	}
	output.WriteString("```")

	return ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, output.String())
}

// ClusterEventsCommand - List recent cluster events
type ClusterEventsCommand struct{}

func (c *ClusterEventsCommand) Name() string                             { return "cluster-events" }
func (c *ClusterEventsCommand) Description() string                      { return "View recent cluster events" }
func (c *ClusterEventsCommand) RequiredPermission() bot.Permission { return bot.PermissionClusterAdmin }

func (c *ClusterEventsCommand) Execute(ctx *CommandContext) error {
	namespace := "game-servers"
	if len(ctx.Args) > 0 {
		namespace = ctx.Args[0]
	}

	k8sClient, err := getKubernetesClient()
	if err != nil {
		return fmt.Errorf("failed to connect to cluster: %v", err)
	}

	events, err := k8sClient.CoreV1().Events(namespace).List(context.Background(), metav1.ListOptions{
		Limit: 20,
	})
	if err != nil {
		return fmt.Errorf("failed to list events: %v", err)
	}

	var output strings.Builder
	output.WriteString(fmt.Sprintf("📅 **Recent Events in `%s`** (last 20)\n", namespace))
	output.WriteString("```\n")

	for _, event := range events.Items {
		age := time.Since(event.LastTimestamp.Time).Round(time.Second)
		output.WriteString(fmt.Sprintf("[%s] %s: %s\n", 
			formatDuration(age), event.Reason, event.Message))
	}
	
	if len(events.Items) == 0 {
		output.WriteString("No recent events\n")
	}
	
	output.WriteString("```")

	return ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, output.String())
}

// ClusterNamespacesCommand - List namespaces
type ClusterNamespacesCommand struct{}

func (c *ClusterNamespacesCommand) Name() string                             { return "cluster-namespaces" }
func (c *ClusterNamespacesCommand) Description() string                      { return "List cluster namespaces" }
func (c *ClusterNamespacesCommand) RequiredPermission() bot.Permission { return bot.PermissionClusterAdmin }

func (c *ClusterNamespacesCommand) Execute(ctx *CommandContext) error {
	k8sClient, err := getKubernetesClient()
	if err != nil {
		return fmt.Errorf("failed to connect to cluster: %v", err)
	}

	namespaces, err := k8sClient.CoreV1().Namespaces().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list namespaces: %v", err)
	}

	var output strings.Builder
	output.WriteString(fmt.Sprintf("📦 **Cluster Namespaces** (%d total)\n", len(namespaces.Items)))
	output.WriteString("```\n")

	for _, ns := range namespaces.Items {
		age := time.Since(ns.CreationTimestamp.Time).Round(time.Second)
		status := string(ns.Status.Phase)
		output.WriteString(fmt.Sprintf("%-30s %-15s %s\n", ns.Name, status, formatDuration(age)))
	}
	output.WriteString("```")

	return ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, output.String())
}

// ============================================================================
// SHOP PURCHASE SYSTEM (WTG + GC dual currency)
// ============================================================================

// BuyCommand - Purchase items from shop
type BuyCommand struct{}

func (c *BuyCommand) Name() string                             { return "buy" }
func (c *BuyCommand) Description() string                      { return "Purchase item from shop" }
func (c *BuyCommand) RequiredPermission() bot.Permission { return bot.PermissionUser }

func (c *BuyCommand) Execute(ctx *CommandContext) error {
	if len(ctx.Args) < 1 {
		return fmt.Errorf("usage: buy <item-id|item-name> [quantity]")
	}

	itemIdentifier := ctx.Args[0]
	quantity := 1
	if len(ctx.Args) > 1 {
		fmt.Sscanf(ctx.Args[1], "%d", &quantity)
		if quantity < 1 || quantity > 100 {
			return fmt.Errorf("quantity must be between 1 and 100")
		}
	}

	// Get shop item
	row := ctx.DB.DB().QueryRow(`
		SELECT id, item_name, item_type, description, price, currency_type, bonus_amount
		FROM shop_items 
		WHERE (id::text = $1 OR LOWER(item_name) = LOWER($1)) AND is_active = true
		LIMIT 1
	`, itemIdentifier)

	var itemID int
	var itemName, itemType, description, currencyType string
	var price int
	var bonusAmount int
	
	if err := row.Scan(&itemID, &itemName, &itemType, &description, &price, &currencyType, &bonusAmount); err != nil {
		return fmt.Errorf("item not found or unavailable")
	}

	totalPrice := price * quantity
	totalBonus := bonusAmount * quantity

	// Get user's current balances
	user, err := ctx.DB.GetOrCreateUser(ctx.Message.Author.ID)
	if err != nil {
		return fmt.Errorf("failed to get user: %v", err)
	}

	wtgBalance, gcBalance := ctx.DB.GetUserCurrencies(ctx.Message.Author.ID)

	// Determine payment method
	var paymentCurrency string
	if currencyType == "USD" {
		// WTG purchases (real money items)
		if wtgBalance < totalPrice {
			return fmt.Errorf("insufficient WTG. Required: %d WTG, You have: %d WTG\n💡 Purchase WTG with `shop` command", 
				totalPrice, wtgBalance)
		}
		paymentCurrency = "WTG"
	} else {
		// GC purchases (soft currency items)
		if gcBalance < totalPrice {
			return fmt.Errorf("insufficient GameCredits. Required: %d GC, You have: %d GC\n💡 Earn GC with `credits earn`, `daily`, or `work`", 
				totalPrice, gcBalance)
		}
		paymentCurrency = "GC"
	}

	// Process purchase
	tx, err := ctx.DB.DB().Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %v", err)
	}
	defer tx.Rollback()

	// Deduct currency
	if paymentCurrency == "WTG" {
		_, err = tx.Exec(`UPDATE users SET wtg_coins = wtg_coins - $1 WHERE discord_id = $2`, 
			totalPrice, ctx.Message.Author.ID)
	} else {
		_, err = tx.Exec(`UPDATE users SET credits = credits - $1 WHERE discord_id = $2`, 
			totalPrice, ctx.Message.Author.ID)
	}
	
	if err != nil {
		return fmt.Errorf("failed to deduct currency: %v", err)
	}

	// Apply item effects based on type
	switch itemType {
	case "wtg_package":
		// Add WTG coins + bonus to user
		totalReceived := quantity + totalBonus
		_, err = tx.Exec(`UPDATE users SET wtg_coins = wtg_coins + $1 WHERE discord_id = $2`, 
			totalReceived, ctx.Message.Author.ID)
		if err != nil {
			return fmt.Errorf("failed to add WTG: %v", err)
		}

	case "gc_conversion":
		// Convert WTG to GC (1 WTG = 1000 GC)
		gcAmount := quantity * 1000
		_, err = tx.Exec(`UPDATE users SET credits = credits + $1 WHERE discord_id = $2`, 
			gcAmount, ctx.Message.Author.ID)
		if err != nil {
			return fmt.Errorf("failed to convert to GC: %v", err)
		}

	default:
		// Add to inventory
		_, err = tx.Exec(`
			INSERT INTO user_inventory (discord_id, item_id, quantity)
			VALUES ($1, $2, $3)
			ON CONFLICT (discord_id, item_id) 
			DO UPDATE SET quantity = user_inventory.quantity + $3
		`, ctx.Message.Author.ID, itemID, quantity)
		
		if err != nil {
			return fmt.Errorf("failed to add to inventory: %v", err)
		}
	}

	// Log transaction
	_, err = tx.Exec(`
		INSERT INTO credit_transactions (from_user, to_user, amount, transaction_type, description, currency_type)
		VALUES ($1, 'SHOP', $2, 'purchase', $3, $4)
	`, ctx.Message.Author.ID, totalPrice, fmt.Sprintf("Purchased %dx %s", quantity, itemName), paymentCurrency)
	
	if err != nil {
		return fmt.Errorf("failed to log transaction: %v", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to complete purchase: %v", err)
	}

	// Send confirmation
	var confirmation string
	switch itemType {
	case "wtg_package":
		totalReceived := quantity + totalBonus
		confirmation = fmt.Sprintf("✅ **Purchase Successful!**\n\n"+
			"💰 Received: **%d WTG**", totalReceived)
		if totalBonus > 0 {
			confirmation += fmt.Sprintf(" (includes %d bonus WTG!)", totalBonus)
		}
	case "gc_conversion":
		gcAmount := quantity * 1000
		confirmation = fmt.Sprintf("✅ **Conversion Successful!**\n\n"+
			"Converted: %d WTG → **%d GameCredits**", quantity, gcAmount)
	default:
		confirmation = fmt.Sprintf("✅ **Purchase Successful!**\n\n"+
			"Item: **%s** x%d\n"+
			"Cost: %d %s\n\n"+
			"Added to your inventory! Use `inventory` to view.", 
			itemName, quantity, totalPrice, paymentCurrency)
	}

	return ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, confirmation)
}

// ConvertCommand - Convert WTG to GC
type ConvertCommand struct{}

func (c *ConvertCommand) Name() string                             { return "convert" }
func (c *ConvertCommand) Description() string                      { return "Convert WTG to GameCredits" }
func (c *ConvertCommand) RequiredPermission() bot.Permission { return bot.PermissionUser }

func (c *ConvertCommand) Execute(ctx *CommandContext) error {
	if len(ctx.Args) < 1 {
		return fmt.Errorf("usage: convert <amount-wtg>\n💡 Conversion rate: 1 WTG = 1000 GC")
	}

	var wtgAmount int
	if _, err := fmt.Sscanf(ctx.Args[0], "%d", &wtgAmount); err != nil || wtgAmount < 1 {
		return fmt.Errorf("invalid amount. Must be a positive number")
	}

	wtgBalance, _ := ctx.DB.GetUserCurrencies(ctx.Message.Author.ID)
	
	if wtgBalance < wtgAmount {
		return fmt.Errorf("insufficient WTG. You have: %d WTG\n💡 Purchase WTG with `shop`", wtgBalance)
	}

	gcAmount := wtgAmount * 1000

	// Process conversion
	tx, err := ctx.DB.DB().Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %v", err)
	}
	defer tx.Rollback()

	// Deduct WTG
	_, err = tx.Exec(`UPDATE users SET wtg_coins = wtg_coins - $1 WHERE discord_id = $2`, 
		wtgAmount, ctx.Message.Author.ID)
	if err != nil {
		return fmt.Errorf("failed to deduct WTG: %v", err)
	}

	// Add GC
	_, err = tx.Exec(`UPDATE users SET credits = credits + $1 WHERE discord_id = $2`, 
		gcAmount, ctx.Message.Author.ID)
	if err != nil {
		return fmt.Errorf("failed to add GC: %v", err)
	}

	// Log transaction
	_, err = tx.Exec(`
		INSERT INTO credit_transactions (from_user, to_user, amount, transaction_type, description, currency_type)
		VALUES ($1, $2, $3, 'conversion', 'WTG to GC conversion', 'BOTH')
	`, ctx.Message.Author.ID, ctx.Message.Author.ID, gcAmount)
	
	if err != nil {
		return fmt.Errorf("failed to log transaction: %v", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to complete conversion: %v", err)
	}

	return ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, fmt.Sprintf(
		"✅ **Conversion Successful!**\n\n"+
		"Converted: **%d WTG** → **%d GameCredits**\n"+
		"Rate: 1 WTG = 1000 GC",
		wtgAmount, gcAmount))
}

// InventoryCommand - View purchased items
type InventoryCommand struct{}

func (c *InventoryCommand) Name() string                             { return "inventory" }
func (c *InventoryCommand) Description() string                      { return "View your inventory" }
func (c *InventoryCommand) RequiredPermission() bot.Permission { return bot.PermissionUser }

func (c *InventoryCommand) Execute(ctx *CommandContext) error {
	rows, err := ctx.DB.DB().Query(`
		SELECT si.item_name, si.item_type, ui.quantity, ui.purchased_at
		FROM user_inventory ui
		JOIN shop_items si ON ui.item_id = si.id
		WHERE ui.discord_id = $1
		ORDER BY ui.purchased_at DESC
	`, ctx.Message.Author.ID)
	
	if err != nil {
		return fmt.Errorf("failed to fetch inventory: %v", err)
	}
	defer rows.Close()

	var output strings.Builder
	output.WriteString("🎒 **Your Inventory**\n━━━━━━━━━━━━━━━━━━━━\n")

	count := 0
	for rows.Next() {
		var itemName, itemType string
		var quantity int
		var purchasedAt time.Time
		
		if err := rows.Scan(&itemName, &itemType, &quantity, &purchasedAt); err != nil {
			continue
		}

		output.WriteString(fmt.Sprintf("**%s** x%d\n", itemName, quantity))
		output.WriteString(fmt.Sprintf("  Type: %s | Purchased: %s\n\n", 
			itemType, purchasedAt.Format("2006-01-02")))
		count++
	}

	if count == 0 {
		output.WriteString("*Your inventory is empty*\n\n💡 Browse items with `shop`")
	}

	return ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, output.String())
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

func getKubernetesClient() (*kubernetes.Clientset, error) {
	var config *rest.Config
	var err error

	// Try in-cluster config first
	config, err = rest.InClusterConfig()
	if err != nil {
		// Fall back to kubeconfig
		config, err = clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
		if err != nil {
			return nil, fmt.Errorf("failed to create kubernetes config: %v", err)
		}
	}

	return kubernetes.NewForConfig(config)
}

func formatDurationV1_6(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	return fmt.Sprintf("%dd", int(d.Hours()/24))
}
