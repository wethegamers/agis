package commands

import (
	"strings"
	"testing"

	"agis-bot/internal/services"

	"github.com/stretchr/testify/require"
)

func TestHandleCreate(t *testing.T) {
	service := services.NewABTestingService()
	cmd := NewExperimentCommand(service)

	msg, err := cmd.HandleCreate("admin", []string{"exp-1", "Reward", "100", "7", "1.0", "1.5"})
	require.NoError(t, err)
	require.True(t, strings.Contains(msg, "Experiment created"))
	require.Len(t, service.ListExperiments(), 1)

	_, err = cmd.HandleCreate("admin", []string{"missing", "params"})
	require.Error(t, err)
}

func TestHandleStartStopAndResults(t *testing.T) {
	service := services.NewABTestingService()
	cmd := NewExperimentCommand(service)

	_, err := cmd.HandleStart("admin", "nonexistent")
	require.Error(t, err)

	_, err = cmd.HandleCreate("admin", []string{"exp-2", "Reward", "100", "7", "1.0", "1.5"})
	require.NoError(t, err)

	msg, err := cmd.HandleStart("admin", "exp-2")
	require.NoError(t, err)
	require.Contains(t, msg, "now running")

	variant, err := service.GetVariant("user-001", "exp-2")
	require.NoError(t, err)
	require.NotNil(t, variant)

	require.NoError(t, service.RecordEvent("user-001", "exp-2", "conversion", 1))
	require.NoError(t, service.RecordEvent("user-001", "exp-2", "reward", 100))

	resultsMsg, err := cmd.HandleResults("admin", "exp-2")
	require.NoError(t, err)
	require.Contains(t, resultsMsg, "Experiment Results")

	stopMsg, err := cmd.HandleStop("admin", "exp-2")
	require.NoError(t, err)
	require.Contains(t, stopMsg, "has been stopped")
}

func TestHandleList(t *testing.T) {
	service := services.NewABTestingService()
	cmd := NewExperimentCommand(service)


	msg, err := cmd.HandleList("admin")
	require.NoError(t, err)
	require.Contains(t, msg, "No experiments")

	_, err = cmd.HandleCreate("admin", []string{"exp-3", "Reward", "50", "3", "1.0", "1.5"})
	require.NoError(t, err)

	require.NoError(t, service.UpdateExperimentStatus("exp-3", "running"))

	msg, err = cmd.HandleList("admin")
	require.NoError(t, err)
	require.Contains(t, msg, "Active Experiments")
}
