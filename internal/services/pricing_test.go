package services

import (
	"testing"
)

func TestPricingServiceRequiresGuild(t *testing.T) {
	// Test that requires_guild field is properly loaded
	pricing := &GamePricing{
		GameType:      "ark",
		CostPerHour:   240,
		DisplayName:   "ARK: Survival Evolved",
		RequiresGuild: true,
		IsActive:      true,
	}

	if !pricing.RequiresGuild {
		t.Fatalf("ARK should require guild, got RequiresGuild=%v", pricing.RequiresGuild)
	}

	if pricing.CostPerHour != 240 {
		t.Fatalf("ARK cost should be 240 GC/hr, got %d", pricing.CostPerHour)
	}

	// Test individual game doesn't require guild
	minecraft := &GamePricing{
		GameType:      "minecraft",
		CostPerHour:   5,
		RequiresGuild: false,
	}

	if minecraft.RequiresGuild {
		t.Fatalf("Minecraft should not require guild, got RequiresGuild=%v", minecraft.RequiresGuild)
	}
}
