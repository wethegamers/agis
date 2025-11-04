package services

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	allocationv1 "agones.dev/agones/pkg/apis/allocation/v1"
	agonesclientset "agones.dev/agones/pkg/client/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// AgonesService manages interaction with Agones GameServers
type AgonesService struct {
	kubeClient   kubernetes.Interface
	agonesClient agonesclientset.Interface
	namespace    string
}

// GameServerInfo represents the current state of a GameServer in Kubernetes
type GameServerInfo struct {
	Name      string
	UID       string
	Status    agonesv1.GameServerState
	Address   string
	Port      int32
	CreatedAt time.Time
	ReadyAt   *time.Time
	GameType  string
	UserID    string
}

// NewAgonesService creates a new Agones service
func NewAgonesService() (*AgonesService, error) {
	var config *rest.Config
	var err error

	// Try to get Kubernetes config
	if os.Getenv("KUBERNETES_SERVICE_HOST") != "" {
		// Running inside cluster
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to get in-cluster config: %v", err)
		}
	} else {
		// Try to use local kubeconfig
		loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
		configOverrides := &clientcmd.ConfigOverrides{}
		kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
		config, err = kubeConfig.ClientConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to load Kubernetes config: %v", err)
		}
	}

	// Create Kubernetes client
	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client: %v", err)
	}

	// Create Agones client
	agonesClient, err := agonesclientset.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Agones client: %v", err)
	}

	// Get namespace from environment or default to agones-dev for development
	namespace := os.Getenv("AGONES_NAMESPACE")
	if namespace == "" {
		namespace = "agones-dev"
	}

	return &AgonesService{
		kubeClient:   kubeClient,
		agonesClient: agonesClient,
		namespace:    namespace,
	}, nil
}

// AllocateGameServer allocates a new GameServer from a Fleet
func (a *AgonesService) AllocateGameServer(ctx context.Context, gameType, serverName, userID string) (*GameServerInfo, error) {
	// Determine which fleet to use based on game type
	fleetName := a.getFleetName(gameType)

	// Create GameServerAllocation
	allocation := &allocationv1.GameServerAllocation{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("%s-", serverName),
			Namespace:    a.namespace,
		},
		Spec: allocationv1.GameServerAllocationSpec{
			Selectors: []allocationv1.GameServerSelector{
				{
					LabelSelector: metav1.LabelSelector{
						MatchLabels: map[string]string{
							"agones.dev/fleet": fleetName,
						},
					},
				},
			},
			// Metadata to be applied to allocated GameServer
			MetaPatch: allocationv1.MetaPatch{
				Labels: map[string]string{
					"wtg.cluster/user-id":     userID,
					"wtg.cluster/server-name": serverName,
					"wtg.cluster/allocated":   "true",
				},
				Annotations: map[string]string{
					"wtg.cluster/allocated-at": time.Now().Format(time.RFC3339),
					"wtg.cluster/allocated-by": "agis-bot",
				},
			},
		},
	}

	// Create the allocation
	result, err := a.agonesClient.AllocationV1().GameServerAllocations(a.namespace).Create(ctx, allocation, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to allocate GameServer: %v", err)
	}

	if result.Status.State != allocationv1.GameServerAllocationAllocated {
		return nil, fmt.Errorf("GameServer allocation failed: %s", result.Status.State)
	}

	// Get the allocated GameServer details from allocation status
	if result.Status.GameServerName == "" {
		return nil, fmt.Errorf("no GameServer name in allocation result")
	}

	// Get the full GameServer object to get the UID and creation time
	gameServer, err := a.agonesClient.AgonesV1().GameServers(a.namespace).Get(ctx, result.Status.GameServerName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get allocated GameServer: %v", err)
	}

	var port int32
	if len(result.Status.Ports) > 0 {
		port = result.Status.Ports[0].Port
	}

	return &GameServerInfo{
		Name:      result.Status.GameServerName,
		UID:       string(gameServer.ObjectMeta.UID),
		Status:    gameServer.Status.State,
		Address:   result.Status.Address,
		Port:      port,
		CreatedAt: gameServer.ObjectMeta.CreationTimestamp.Time,
		GameType:  gameType,
		UserID:    userID,
	}, nil
}

// GetGameServerByUID retrieves GameServer information by UID
func (a *AgonesService) GetGameServerByUID(ctx context.Context, uid string) (*GameServerInfo, error) {
	// List all GameServers and find by UID
	gameServers, err := a.agonesClient.AgonesV1().GameServers(a.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list GameServers: %v", err)
	}

	for _, gs := range gameServers.Items {
		if string(gs.UID) == uid {
			return a.gameServerToInfo(&gs), nil
		}
	}

	return nil, fmt.Errorf("GameServer with UID %s not found", uid)
}

// GetGameServerByName retrieves GameServer information by name
func (a *AgonesService) GetGameServerByName(ctx context.Context, name string) (*GameServerInfo, error) {
	gameServer, err := a.agonesClient.AgonesV1().GameServers(a.namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get GameServer %s: %v", name, err)
	}

	return a.gameServerToInfo(gameServer), nil
}

// GetUserGameServers retrieves all GameServers for a specific user
func (a *AgonesService) GetUserGameServers(ctx context.Context, userID string) ([]*GameServerInfo, error) {
	gameServers, err := a.agonesClient.AgonesV1().GameServers(a.namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("wtg.cluster/user-id=%s", userID),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list user GameServers: %v", err)
	}

	var servers []*GameServerInfo
	for _, gs := range gameServers.Items {
		servers = append(servers, a.gameServerToInfo(&gs))
	}

	return servers, nil
}

// DeleteGameServer deletes a GameServer by name
func (a *AgonesService) DeleteGameServer(ctx context.Context, name string) error {
	return a.agonesClient.AgonesV1().GameServers(a.namespace).Delete(ctx, name, metav1.DeleteOptions{})
}

// DeleteGameServerByUID deletes a GameServer by UID
func (a *AgonesService) DeleteGameServerByUID(ctx context.Context, uid string) error {
	// First get the GameServer to find its name
	gsInfo, err := a.GetGameServerByUID(ctx, uid)
	if err != nil {
		return fmt.Errorf("failed to find GameServer with UID %s: %v", uid, err)
	}

	// Delete by name
	return a.DeleteGameServer(ctx, gsInfo.Name)
}

// GetGameServerStatus checks the current status of a GameServer
func (a *AgonesService) GetGameServerStatus(ctx context.Context, uid string) (*GameServerInfo, error) {
	return a.GetGameServerByUID(ctx, uid)
}

// WatchGameServerStatus watches for changes to a specific GameServer
func (a *AgonesService) WatchGameServerStatus(ctx context.Context, uid string, callback func(*GameServerInfo)) error {
	// This would implement a watcher, but for now we'll use polling
	// In a production system, you'd want to use Kubernetes watch API
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			info, err := a.GetGameServerByUID(ctx, uid)
			if err != nil {
				log.Printf("Error watching GameServer %s: %v", uid, err)
				continue
			}
			callback(info)
		}
	}
}

// Helper methods

// FindGameServerByServerName finds a GameServer by our label wtg.cluster/server-name
func (a *AgonesService) FindGameServerByServerName(ctx context.Context, serverName string) (*GameServerInfo, error) {
	selector := fmt.Sprintf("wtg.cluster/server-name=%s", serverName)
	gsList, err := a.agonesClient.AgonesV1().GameServers(a.namespace).List(ctx, metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		return nil, fmt.Errorf("failed to list GameServers: %v", err)
	}
	if len(gsList.Items) == 0 {
		return nil, fmt.Errorf("no GameServer found for %s", serverName)
	}
	gs := gsList.Items[0]
	info := a.gameServerToInfo(&gs)
	return info, nil
}

func (a *AgonesService) getFleetName(gameType string) string {
	switch gameType {
	case "minecraft":
		// Use the development test fleet; adjust per environment via Vault/secret if needed
		return "agis-dev-fleet"
	case "cs2":
		return "wtg-premium-fleet"
	case "terraria":
		return "wtg-free-fleet"
	case "gmod":
		return "wtg-premium-fleet"
	default:
		return "wtg-free-fleet"
	}
}

func (a *AgonesService) gameServerToInfo(gs *agonesv1.GameServer) *GameServerInfo {
	info := &GameServerInfo{
		Name:      gs.Name,
		UID:       string(gs.UID),
		Status:    gs.Status.State,
		CreatedAt: gs.CreationTimestamp.Time,
		GameType:  gs.Labels["wtg.cluster/game-type"],
		UserID:    gs.Labels["wtg.cluster/user-id"],
	}

	// Set address and port if available
	if gs.Status.Address != "" && len(gs.Status.Ports) > 0 {
		info.Address = gs.Status.Address
		info.Port = gs.Status.Ports[0].Port
	}

	// Set ready time if status is Ready or Allocated
	if gs.Status.State == agonesv1.GameServerStateReady || gs.Status.State == agonesv1.GameServerStateAllocated {
		// For now, estimate ready time; in production you'd track this via events
		readyTime := gs.CreationTimestamp.Add(2 * time.Minute)
		info.ReadyAt = &readyTime
	}

	return info
}

// GetGameServerConnection returns connection info for a GameServer
func (a *AgonesService) GetGameServerConnection(ctx context.Context, uid string) (string, int32, error) {
	info, err := a.GetGameServerByUID(ctx, uid)
	if err != nil {
		return "", 0, err
	}

	if info.Status != agonesv1.GameServerStateReady && info.Status != agonesv1.GameServerStateAllocated {
		return "", 0, fmt.Errorf("GameServer is not ready yet (status: %s)", info.Status)
	}

	return info.Address, info.Port, nil
}
