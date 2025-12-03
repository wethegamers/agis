package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestNamespace(t *testing.T) {
	if Namespace != "agis" {
		t.Errorf("expected namespace 'agis', got %s", Namespace)
	}
}

func TestCommandsTotal(t *testing.T) {
	CommandsTotal.WithLabelValues("test", "success").Inc()

	value := testutil.ToFloat64(CommandsTotal.WithLabelValues("test", "success"))
	if value < 1 {
		t.Errorf("expected counter >= 1, got %f", value)
	}
}

func TestCommandDuration(t *testing.T) {
	CommandDuration.WithLabelValues("test").Observe(0.5)

	// Just verify it doesn't panic
	if CommandDuration == nil {
		t.Error("expected non-nil histogram")
	}
}

func TestGameServersTotal(t *testing.T) {
	GameServersTotal.WithLabelValues("minecraft", "running").Set(5)

	value := testutil.ToFloat64(GameServersTotal.WithLabelValues("minecraft", "running"))
	if value != 5 {
		t.Errorf("expected gauge 5, got %f", value)
	}
}

func TestServerOperations(t *testing.T) {
	ServerOperations.WithLabelValues("create", "minecraft", "success").Inc()

	value := testutil.ToFloat64(ServerOperations.WithLabelValues("create", "minecraft", "success"))
	if value < 1 {
		t.Errorf("expected counter >= 1, got %f", value)
	}
}

func TestCreditsTransactions(t *testing.T) {
	CreditsTransactions.WithLabelValues("purchase").Inc()

	value := testutil.ToFloat64(CreditsTransactions.WithLabelValues("purchase"))
	if value < 1 {
		t.Errorf("expected counter >= 1, got %f", value)
	}
}

func TestActiveUsers(t *testing.T) {
	ActiveUsers.Set(100)

	value := testutil.ToFloat64(ActiveUsers)
	if value != 100 {
		t.Errorf("expected gauge 100, got %f", value)
	}
}

func TestUsersByTier(t *testing.T) {
	UsersByTier.WithLabelValues("premium").Set(50)

	value := testutil.ToFloat64(UsersByTier.WithLabelValues("premium"))
	if value != 50 {
		t.Errorf("expected gauge 50, got %f", value)
	}
}

func TestDatabaseConnections(t *testing.T) {
	DatabaseConnections.Set(10)

	value := testutil.ToFloat64(DatabaseConnections)
	if value != 10 {
		t.Errorf("expected gauge 10, got %f", value)
	}
}

func TestAPIRequestsTotal(t *testing.T) {
	APIRequestsTotal.WithLabelValues("GET", "/api/users", "200").Inc()

	value := testutil.ToFloat64(APIRequestsTotal.WithLabelValues("GET", "/api/users", "200"))
	if value < 1 {
		t.Errorf("expected counter >= 1, got %f", value)
	}
}

func TestSetBuildInfo(t *testing.T) {
	SetBuildInfo("1.0.0", "abc123", "2024-01-01")

	value := testutil.ToFloat64(BuildInfo.WithLabelValues("1.0.0", "abc123", "2024-01-01"))
	if value != 1 {
		t.Errorf("expected build info 1, got %f", value)
	}
}
