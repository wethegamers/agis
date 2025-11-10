package commands

import (
	"fmt"
	"time"

	"agis-bot/internal/services"
)

// ExperimentCommand handles A/B testing experiment management
type ExperimentCommand struct {
	abService *services.ABTestingService
}

// NewExperimentCommand creates a new experiment command handler
func NewExperimentCommand(abService *services.ABTestingService) *ExperimentCommand {
	return &ExperimentCommand{
		abService: abService,
	}
}

// HandleCreate creates a new A/B test experiment
// Usage: /experiment create <id> <name> <traffic%> <duration_days> <variant1_id>:<multiplier> <variant2_id>:<multiplier>
func (c *ExperimentCommand) HandleCreate(userID string, args []string) (string, error) {
	if len(args) < 6 {
		return "", fmt.Errorf("usage: /experiment create <id> <name> <traffic%%> <duration_days> <control_multiplier> <variant_multiplier>")
	}

	experimentID := args[0]
	name := args[1]
	
	var trafficAlloc float64
	fmt.Sscanf(args[2], "%f", &trafficAlloc)
	trafficAlloc /= 100.0 // Convert percentage to 0.0-1.0

	var durationDays int
	fmt.Sscanf(args[3], "%d", &durationDays)

	var controlMultiplier, variantMultiplier float64
	fmt.Sscanf(args[4], "%f", &controlMultiplier)
	fmt.Sscanf(args[5], "%f", &variantMultiplier)

	experiment := &services.ExperimentConfig{
		ID:           experimentID,
		Name:         name,
		Description:  fmt.Sprintf("Reward multiplier test: control %.1fx vs variant %.1fx", controlMultiplier, variantMultiplier),
		StartDate:    time.Now(),
		EndDate:      time.Now().Add(time.Duration(durationDays) * 24 * time.Hour),
		TrafficAlloc: trafficAlloc,
		TargetMetric: "conversion_rate",
		Status:       "draft",
		Variants: []services.Variant{
			{
				ID:          "control",
				Name:        "Control",
				Allocation:  0.5,
				Config:      map[string]interface{}{"multiplier": controlMultiplier},
				Description: fmt.Sprintf("Control group with %.1fx multiplier", controlMultiplier),
			},
			{
				ID:          "variant_a",
				Name:        "Variant A",
				Allocation:  0.5,
				Config:      map[string]interface{}{"multiplier": variantMultiplier},
				Description: fmt.Sprintf("Test group with %.1fx multiplier", variantMultiplier),
			},
		},
	}

	if err := c.abService.CreateExperiment(experiment); err != nil {
		return "", fmt.Errorf("failed to create experiment: %w", err)
	}

	return fmt.Sprintf("✅ Experiment created: **%s**\n"+
		"ID: `%s`\n"+
		"Traffic: %.0f%%\n"+
		"Duration: %d days\n"+
		"Control: %.1fx | Variant: %.1fx\n"+
		"Status: **draft**\n\n"+
		"Run `/experiment start %s` to activate",
		name, experimentID, trafficAlloc*100, durationDays, controlMultiplier, variantMultiplier, experimentID), nil
}

// HandleStart starts a draft experiment
func (c *ExperimentCommand) HandleStart(userID string, experimentID string) (string, error) {
	if err := c.abService.UpdateExperimentStatus(experimentID, "running"); err != nil {
		return "", fmt.Errorf("failed to start experiment: %w", err)
	}

	return fmt.Sprintf("🚀 Experiment **%s** is now running!\nUsers will be automatically assigned to variants.", experimentID), nil
}

// HandleStop stops a running experiment
func (c *ExperimentCommand) HandleStop(userID string, experimentID string) (string, error) {
	if err := c.abService.UpdateExperimentStatus(experimentID, "completed"); err != nil {
		return "", fmt.Errorf("failed to stop experiment: %w", err)
	}

	return fmt.Sprintf("🛑 Experiment **%s** has been stopped.\nResults are now final.", experimentID), nil
}

// HandleResults shows experiment results
func (c *ExperimentCommand) HandleResults(userID string, experimentID string) (string, error) {
	results, err := c.abService.GetExperimentResults(experimentID)
	if err != nil {
		return "", fmt.Errorf("failed to get results: %w", err)
	}

	if len(results) == 0 {
		return fmt.Sprintf("No results yet for experiment **%s**", experimentID), nil
	}

	message := fmt.Sprintf("📊 **Experiment Results: %s**\n\n", experimentID)
	for _, result := range results {
		message += fmt.Sprintf("**%s** (n=%d)\n", result.VariantID, result.SampleSize)
		message += fmt.Sprintf("  Conversion Rate: %.2f%%\n", result.ConversionRate*100)
		message += fmt.Sprintf("  Avg Reward: %.0f GC\n", result.AvgRewardAmount)
		message += fmt.Sprintf("  Revenue/User: %.0f GC\n", result.RevenuePerUser)
		message += fmt.Sprintf("  Fraud Rate: %.2f%%\n\n", result.FraudRate*100)
	}

	// Calculate statistical significance (simple version)
	if len(results) == 2 {
		controlRate := results[0].ConversionRate
		variantRate := results[1].ConversionRate
		uplift := ((variantRate - controlRate) / controlRate) * 100

		message += fmt.Sprintf("📈 **Uplift**: %.2f%%\n", uplift)
		
		if uplift > 5 && results[1].SampleSize > 100 {
			message += "✅ **Recommendation**: Deploy variant (statistically significant)\n"
		} else if uplift < -5 && results[1].SampleSize > 100 {
			message += "❌ **Recommendation**: Keep control (variant performing worse)\n"
		} else {
			message += "⚠️  **Recommendation**: Continue test (not enough data or inconclusive)\n"
		}
	}

	return message, nil
}

// HandleList lists all experiments
func (c *ExperimentCommand) HandleList(userID string) (string, error) {
	experiments := c.abService.ListExperiments()
	
	if len(experiments) == 0 {
		return "No experiments found. Create one with `/experiment create`", nil
	}

	message := "📋 **Active Experiments**\n\n"
	for _, exp := range experiments {
		statusEmoji := map[string]string{
			"draft":     "📝",
			"running":   "🚀",
			"paused":    "⏸️",
			"completed": "✅",
			"archived":  "📦",
		}[exp.Status]

		message += fmt.Sprintf("%s **%s** (`%s`)\n", statusEmoji, exp.Name, exp.ID)
		message += fmt.Sprintf("  Status: %s | Traffic: %.0f%%\n", exp.Status, exp.TrafficAlloc*100)
		message += fmt.Sprintf("  Ends: %s\n\n", exp.EndDate.Format("2006-01-02"))
	}

	return message, nil
}
