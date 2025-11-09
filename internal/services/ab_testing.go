package services

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// ExperimentConfig defines an A/B test experiment
type ExperimentConfig struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	Description     string                 `json:"description"`
	StartDate       time.Time              `json:"start_date"`
	EndDate         time.Time              `json:"end_date"`
	TrafficAlloc    float64                `json:"traffic_alloc"` // 0.0-1.0 (percentage of users)
	Variants        []Variant              `json:"variants"`
	TargetMetric    string                 `json:"target_metric"` // e.g., "conversion_rate", "revenue_per_user"
	Status          string                 `json:"status"`        // "draft", "running", "paused", "completed"
	Metadata        map[string]interface{} `json:"metadata"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

// Variant represents a single variant in an A/B test
type Variant struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`       // e.g., "control", "variant_a", "variant_b"
	Allocation  float64                `json:"allocation"` // 0.0-1.0 (percentage of experiment traffic)
	Config      map[string]interface{} `json:"config"`     // Variant-specific config (e.g., reward multipliers)
	Description string                 `json:"description"`
}

// Assignment tracks which variant a user is assigned to
type Assignment struct {
	UserID       string    `json:"user_id"`
	ExperimentID string    `json:"experiment_id"`
	VariantID    string    `json:"variant_id"`
	AssignedAt   time.Time `json:"assigned_at"`
	Sticky       bool      `json:"sticky"` // If true, user stays in same variant
}

// ExperimentResult stores aggregated metrics for analysis
type ExperimentResult struct {
	ExperimentID     string             `json:"experiment_id"`
	VariantID        string             `json:"variant_id"`
	SampleSize       int                `json:"sample_size"`
	ConversionRate   float64            `json:"conversion_rate"`
	RevenuePerUser   float64            `json:"revenue_per_user"`
	AvgRewardAmount  float64            `json:"avg_reward_amount"`
	FraudRate        float64            `json:"fraud_rate"`
	Metrics          map[string]float64 `json:"metrics"`
	LastUpdated      time.Time          `json:"last_updated"`
}

// ABTestingService manages A/B experiments for reward rates
type ABTestingService struct {
	experiments map[string]*ExperimentConfig
	assignments map[string]*Assignment // key: userID+experimentID
	results     map[string]*ExperimentResult
	mu          sync.RWMutex
	rng         *rand.Rand
}

// NewABTestingService creates a new A/B testing service
func NewABTestingService() *ABTestingService {
	return &ABTestingService{
		experiments: make(map[string]*ExperimentConfig),
		assignments: make(map[string]*Assignment),
		results:     make(map[string]*ExperimentResult),
		rng:         rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// CreateExperiment creates a new A/B test experiment
func (s *ABTestingService) CreateExperiment(config *ExperimentConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Validate config
	if config.ID == "" {
		return errors.New("experiment ID is required")
	}
	if len(config.Variants) < 2 {
		return errors.New("experiment must have at least 2 variants")
	}

	// Validate variant allocations sum to 1.0
	totalAlloc := 0.0
	for _, v := range config.Variants {
		totalAlloc += v.Allocation
	}
	if totalAlloc < 0.99 || totalAlloc > 1.01 {
		return fmt.Errorf("variant allocations must sum to 1.0, got %.2f", totalAlloc)
	}

	config.CreatedAt = time.Now()
	config.UpdatedAt = time.Now()
	s.experiments[config.ID] = config

	// Initialize result tracking for each variant
	for _, variant := range config.Variants {
		resultKey := fmt.Sprintf("%s:%s", config.ID, variant.ID)
		s.results[resultKey] = &ExperimentResult{
			ExperimentID: config.ID,
			VariantID:    variant.ID,
			Metrics:      make(map[string]float64),
			LastUpdated:  time.Now(),
		}
	}

	return nil
}

// GetVariant returns the assigned variant for a user
// If user not yet assigned, assigns them based on experiment config
func (s *ABTestingService) GetVariant(userID, experimentID string) (*Variant, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	experiment, ok := s.experiments[experimentID]
	if !ok {
		return nil, fmt.Errorf("experiment %s not found", experimentID)
	}

	// Check if experiment is active
	now := time.Now()
	if experiment.Status != "running" {
		return nil, fmt.Errorf("experiment %s is not running (status: %s)", experimentID, experiment.Status)
	}
	if now.Before(experiment.StartDate) || now.After(experiment.EndDate) {
		return nil, fmt.Errorf("experiment %s is not active in current time window", experimentID)
	}

	// Check for existing assignment
	assignmentKey := fmt.Sprintf("%s:%s", userID, experimentID)
	if assignment, ok := s.assignments[assignmentKey]; ok {
		// Find variant by ID
		for i := range experiment.Variants {
			if experiment.Variants[i].ID == assignment.VariantID {
				return &experiment.Variants[i], nil
			}
		}
	}

	// Check traffic allocation (not all users enter experiment)
	userHash := hashString(userID + experimentID)
	userHashFloat := float64(userHash%10000) / 10000.0
	if userHashFloat > experiment.TrafficAlloc {
		// User not in experiment - return nil (use default behavior)
		return nil, nil
	}

	// Assign user to variant using deterministic hash
	variantIndex := s.assignVariant(userID, experiment)
	assignedVariant := &experiment.Variants[variantIndex]

	// Store assignment
	s.assignments[assignmentKey] = &Assignment{
		UserID:       userID,
		ExperimentID: experimentID,
		VariantID:    assignedVariant.ID,
		AssignedAt:   now,
		Sticky:       true,
	}

	return assignedVariant, nil
}

// assignVariant deterministically assigns a user to a variant based on hash
func (s *ABTestingService) assignVariant(userID string, experiment *ExperimentConfig) int {
	// Use hash for deterministic assignment (same user always gets same variant)
	userHash := hashString(userID + experiment.ID + "variant")
	hashFloat := float64(userHash%10000) / 10000.0

	// Map hash to variant based on allocations
	cumulative := 0.0
	for i, variant := range experiment.Variants {
		cumulative += variant.Allocation
		if hashFloat < cumulative {
			return i
		}
	}

	// Fallback to last variant
	return len(experiment.Variants) - 1
}

// RecordEvent records an event for experiment analysis
func (s *ABTestingService) RecordEvent(userID, experimentID string, eventType string, value float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Find user's assignment
	assignmentKey := fmt.Sprintf("%s:%s", userID, experimentID)
	assignment, ok := s.assignments[assignmentKey]
	if !ok {
		// User not in experiment, skip tracking
		return nil
	}

	// Update result metrics
	resultKey := fmt.Sprintf("%s:%s", experimentID, assignment.VariantID)
	result, ok := s.results[resultKey]
	if !ok {
		return fmt.Errorf("result not found for %s", resultKey)
	}

	result.SampleSize++
	switch eventType {
	case "conversion":
		result.ConversionRate = (result.ConversionRate*float64(result.SampleSize-1) + value) / float64(result.SampleSize)
	case "revenue":
		result.RevenuePerUser += value
	case "reward":
		result.AvgRewardAmount = (result.AvgRewardAmount*float64(result.SampleSize-1) + value) / float64(result.SampleSize)
	case "fraud":
		result.FraudRate = (result.FraudRate*float64(result.SampleSize-1) + value) / float64(result.SampleSize)
	default:
		// Custom metric
		result.Metrics[eventType] = (result.Metrics[eventType]*float64(result.SampleSize-1) + value) / float64(result.SampleSize)
	}

	result.LastUpdated = time.Now()
	return nil
}

// GetExperimentResults returns aggregated results for an experiment
func (s *ABTestingService) GetExperimentResults(experimentID string) ([]*ExperimentResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	experiment, ok := s.experiments[experimentID]
	if !ok {
		return nil, fmt.Errorf("experiment %s not found", experimentID)
	}

	results := make([]*ExperimentResult, 0, len(experiment.Variants))
	for _, variant := range experiment.Variants {
		resultKey := fmt.Sprintf("%s:%s", experimentID, variant.ID)
		if result, ok := s.results[resultKey]; ok {
			results = append(results, result)
		}
	}

	return results, nil
}

// ListExperiments returns all experiments
func (s *ABTestingService) ListExperiments() []*ExperimentConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()

	experiments := make([]*ExperimentConfig, 0, len(s.experiments))
	for _, exp := range s.experiments {
		experiments = append(experiments, exp)
	}
	return experiments
}

// UpdateExperimentStatus updates experiment status
func (s *ABTestingService) UpdateExperimentStatus(experimentID, status string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	experiment, ok := s.experiments[experimentID]
	if !ok {
		return fmt.Errorf("experiment %s not found", experimentID)
	}

	validStatuses := map[string]bool{"draft": true, "running": true, "paused": true, "completed": true}
	if !validStatuses[status] {
		return fmt.Errorf("invalid status: %s", status)
	}

	experiment.Status = status
	experiment.UpdatedAt = time.Now()
	return nil
}

// Helper: Hash string to uint32 for deterministic assignment
func hashString(s string) uint32 {
	hash := md5.Sum([]byte(s))
	hexStr := hex.EncodeToString(hash[:])
	// Use first 8 chars of hex as uint32
	var result uint32
	fmt.Sscanf(hexStr[:8], "%x", &result)
	return result
}
