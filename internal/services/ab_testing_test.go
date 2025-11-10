package services

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type experimentOption func(*ExperimentConfig)

func newTestExperiment(id string, opts ...experimentOption) *ExperimentConfig {
	now := time.Now()
	experiment := &ExperimentConfig{
		ID:           id,
		Name:         "Test Experiment",
		Description:  "Test description",
		StartDate:    now.Add(-time.Minute),
		EndDate:      now.Add(time.Hour),
		TrafficAlloc: 1.0,
		Variants: []Variant{
			{
				ID:         "control",
				Name:       "Control",
				Allocation: 0.5,
				Config:     map[string]interface{}{"multiplier": 1.0},
			},
			{
				ID:         "variant_a",
				Name:       "Variant A",
				Allocation: 0.5,
				Config:     map[string]interface{}{"multiplier": 1.5},
			},
		},
		TargetMetric: "conversion_rate",
		Status:       "draft",
	}

	for _, opt := range opts {
		opt(experiment)
	}

	return experiment
}

func withTrafficAlloc(alloc float64) experimentOption {
	return func(exp *ExperimentConfig) {
		exp.TrafficAlloc = alloc
	}
}

func TestABTestingServiceLifecycle(t *testing.T) {
	t.Run("complete lifecycle records results", func(t *testing.T) {
		service := NewABTestingService()
		experiment := newTestExperiment("reward-test")

		require.NoError(t, service.CreateExperiment(experiment))
		require.NoError(t, service.UpdateExperimentStatus(experiment.ID, "running"))

		variant, err := service.GetVariant("user-001", experiment.ID)
		require.NoError(t, err)
		require.NotNil(t, variant)

		variantAgain, err := service.GetVariant("user-001", experiment.ID)
		require.NoError(t, err)
		require.Equal(t, variant.ID, variantAgain.ID, "assignment should be sticky")

		otherVariant, err := service.GetVariant("user-002", experiment.ID)
		require.NoError(t, err)
		require.NotNil(t, otherVariant)
		require.Contains(t, []string{"control", "variant_a"}, otherVariant.ID)

		require.NoError(t, service.RecordEvent("user-001", experiment.ID, "conversion", 1.0))
		require.NoError(t, service.RecordEvent("user-001", experiment.ID, "reward", 120.0))
		require.NoError(t, service.RecordEvent("user-001", experiment.ID, "custom_metric", 3.5))

		results, err := service.GetExperimentResults(experiment.ID)
		require.NoError(t, err)
		require.Len(t, results, 2)

		var assignedResult *ExperimentResult
		for _, res := range results {
			if res.VariantID == variant.ID {
				assignedResult = res
				break
			}
		}

		require.NotNil(t, assignedResult, "assigned variant should have results")
		require.Equal(t, 3, assignedResult.SampleSize)
		require.InDelta(t, 1.0, assignedResult.ConversionRate, 1e-6)
		require.InDelta(t, 60.0, assignedResult.AvgRewardAmount, 1e-6)
		require.InDelta(t, 3.5/3.0, assignedResult.Metrics["custom_metric"], 1e-6)
		require.False(t, assignedResult.LastUpdated.IsZero())

		require.NoError(t, service.RecordEvent("unassigned-user", experiment.ID, "conversion", 1.0))
		require.Equal(t, 3, assignedResult.SampleSize, "unassigned users should not change results")
	})
}

func TestABTestingServiceTrafficAllocation(t *testing.T) {
	service := NewABTestingService()
	experiment := newTestExperiment("limited-traffic", withTrafficAlloc(0.0))

	require.NoError(t, service.CreateExperiment(experiment))
	require.NoError(t, service.UpdateExperimentStatus(experiment.ID, "running"))

	variant, err := service.GetVariant("user-traffic-excluded", experiment.ID)
	require.NoError(t, err)
	require.Nil(t, variant)
}

func TestABTestingServiceStatusValidation(t *testing.T) {
	service := NewABTestingService()
	experiment := newTestExperiment("status-test")

	require.NoError(t, service.CreateExperiment(experiment))

	_, err := service.GetVariant("user", experiment.ID)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not running")

	require.Error(t, service.UpdateExperimentStatus(experiment.ID, "invalid"))
}
