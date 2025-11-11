package services

import (
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"
)

// GamePricing represents the cost structure for a game type
type GamePricing struct {
	GameType      string
	CostPerHour   int       // Cost in GameCredits
	DisplayName   string    // User-friendly name
	Description   string    // Brief description
	IsActive      bool      // Whether this game is available
	RequiresGuild bool      // Whether this game requires guild treasury (Titan-tier)
	MinCredits    int       // Minimum credits required to create
	UpdatedAt     time.Time // Last time pricing was updated
}

// PricingService manages dynamic game pricing from database
type PricingService struct {
	db         *sql.DB
	cache      map[string]*GamePricing
	cacheMutex sync.RWMutex
	lastSync   time.Time
}

// NewPricingService creates a new pricing service
func NewPricingService(db *sql.DB) (*PricingService, error) {
	service := &PricingService{
		db:    db,
		cache: make(map[string]*GamePricing),
	}

	// Initialize pricing table
	if err := service.initPricingTable(); err != nil {
		return nil, fmt.Errorf("failed to initialize pricing table: %v", err)
	}

	// Load initial pricing data
	if err := service.syncPricing(); err != nil {
		return nil, fmt.Errorf("failed to sync initial pricing: %v", err)
	}

	log.Printf("💰 Pricing service initialized with %d game types", len(service.cache))
	return service, nil
}

func (p *PricingService) initPricingTable() error {
	createTable := `
	CREATE TABLE IF NOT EXISTS game_pricing (
		game_type VARCHAR(50) PRIMARY KEY,
		cost_per_hour INTEGER NOT NULL,
		display_name VARCHAR(100) NOT NULL,
		description TEXT DEFAULT '',
		is_active BOOLEAN DEFAULT true,
		requires_guild BOOLEAN DEFAULT false,
		min_credits INTEGER NOT NULL,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`

	_, err := p.db.Exec(createTable)
	if err != nil {
		return err
	}

	// Seed initial pricing (matches Economy Plan v2.0)
	// CRITICAL: These are REAL costs based on actual infrastructure
	seedPricing := `
	INSERT INTO game_pricing (game_type, cost_per_hour, display_name, description, is_active, requires_guild, min_credits)
	VALUES 
		('minecraft', 5, 'Minecraft Java Edition', 'Vanilla or modded Minecraft server', true, false, 5),
		('cs2', 8, 'Counter-Strike 2', 'CS2 dedicated server', true, false, 8),
		('terraria', 3, 'Terraria', 'Multiplayer Terraria world', true, false, 3),
		('gmod', 6, 'Garry''s Mod', 'GMod server with addons', true, false, 6),
		('ark', 240, 'ARK: Survival Evolved', 'TITAN-TIER: Requires guild pooling', true, true, 240)
	ON CONFLICT (game_type) DO NOTHING
	`

	_, err = p.db.Exec(seedPricing)
	return err
}

// syncPricing reloads pricing from database
func (p *PricingService) syncPricing() error {
	rows, err := p.db.Query(`
		SELECT game_type, cost_per_hour, display_name, description, is_active, requires_guild, min_credits, updated_at
		FROM game_pricing
		WHERE is_active = true
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	p.cacheMutex.Lock()
	defer p.cacheMutex.Unlock()

	newCache := make(map[string]*GamePricing)
	for rows.Next() {
		pricing := &GamePricing{}
		err := rows.Scan(
			&pricing.GameType,
			&pricing.CostPerHour,
			&pricing.DisplayName,
			&pricing.Description,
			&pricing.IsActive,
			&pricing.RequiresGuild,
			&pricing.MinCredits,
			&pricing.UpdatedAt,
		)
		if err != nil {
			log.Printf("Error scanning pricing row: %v", err)
			continue
		}
		newCache[pricing.GameType] = pricing
	}

	p.cache = newCache
	p.lastSync = time.Now()

	log.Printf("💰 Synced pricing for %d active games", len(newCache))
	return nil
}

// GetPricing returns pricing for a specific game type
func (p *PricingService) GetPricing(gameType string) (*GamePricing, error) {
	// Auto-sync if cache is older than 5 minutes
	if time.Since(p.lastSync) > 5*time.Minute {
		if err := p.syncPricing(); err != nil {
			log.Printf("Warning: Failed to sync pricing: %v", err)
		}
	}

	p.cacheMutex.RLock()
	defer p.cacheMutex.RUnlock()

	pricing, exists := p.cache[gameType]
	if !exists {
		return nil, fmt.Errorf("game type '%s' not found or inactive", gameType)
	}

	return pricing, nil
}

// GetAllPricing returns all active game pricing
func (p *PricingService) GetAllPricing() []*GamePricing {
	// Auto-sync if cache is older than 5 minutes
	if time.Since(p.lastSync) > 5*time.Minute {
		if err := p.syncPricing(); err != nil {
			log.Printf("Warning: Failed to sync pricing: %v", err)
		}
	}

	p.cacheMutex.RLock()
	defer p.cacheMutex.RUnlock()

	result := make([]*GamePricing, 0, len(p.cache))
	for _, pricing := range p.cache {
		result = append(result, pricing)
	}
	return result
}

// UpdatePricing updates pricing for a game (admin use only)
func (p *PricingService) UpdatePricing(gameType string, costPerHour int, minCredits int) error {
	_, err := p.db.Exec(`
		UPDATE game_pricing
		SET cost_per_hour = $1, min_credits = $2, updated_at = CURRENT_TIMESTAMP
		WHERE game_type = $3
	`, costPerHour, minCredits, gameType)

	if err != nil {
		return err
	}

	// Force immediate cache refresh
	return p.syncPricing()
}

// AddGameType adds a new game type to pricing (admin use only)
func (p *PricingService) AddGameType(gameType, displayName, description string, costPerHour, minCredits int) error {
	_, err := p.db.Exec(`
		INSERT INTO game_pricing (game_type, cost_per_hour, display_name, description, is_active, min_credits)
		VALUES ($1, $2, $3, $4, true, $5)
	`, gameType, costPerHour, displayName, description, minCredits)

	if err != nil {
		return err
	}

	// Force immediate cache refresh
	return p.syncPricing()
}

// DisableGameType deactivates a game type
func (p *PricingService) DisableGameType(gameType string) error {
	_, err := p.db.Exec(`
		UPDATE game_pricing
		SET is_active = false, updated_at = CURRENT_TIMESTAMP
		WHERE game_type = $1
	`, gameType)

	if err != nil {
		return err
	}

	// Force immediate cache refresh
	return p.syncPricing()
}

// IsValidGameType checks if a game type exists and is active
func (p *PricingService) IsValidGameType(gameType string) bool {
	p.cacheMutex.RLock()
	defer p.cacheMutex.RUnlock()

	_, exists := p.cache[gameType]
	return exists
}

// GetCostPerHour returns just the cost for a game type (convenience method)
func (p *PricingService) GetCostPerHour(gameType string) (int, error) {
	pricing, err := p.GetPricing(gameType)
	if err != nil {
		return 0, err
	}
	return pricing.CostPerHour, nil
}

// ValidateUserCanAfford checks if user has enough credits for a game type
func (p *PricingService) ValidateUserCanAfford(gameType string, userCredits int) (bool, int, error) {
	pricing, err := p.GetPricing(gameType)
	if err != nil {
		return false, 0, err
	}

	canAfford := userCredits >= pricing.MinCredits
	return canAfford, pricing.CostPerHour, nil
}
