package agones

import (
	"context"
	"fmt"
	"log"
	"time"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	allocationv1 "agones.dev/agones/pkg/apis/allocation/v1"
	autoscalingv1 "agones.dev/agones/pkg/apis/autoscaling/v1"
	"agones.dev/agones/pkg/client/clientset/versioned"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Client represents an Agones client for managing game servers
type Client struct {
	clientset versioned.Interface
	namespace string
}

// NewClient creates a new Agones client
func NewClient(kubeconfig string, namespace string) (*Client, error) {
	var config *rest.Config
	var err error

	if kubeconfig != "" {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	} else {
		config, err = rest.InClusterConfig()
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes config: %w", err)
	}

	agonesClient, err := versioned.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create agones client: %w", err)
	}

	if namespace == "" {
		namespace = "agones-system"
	}

	return &Client{
		clientset: agonesClient,
		namespace: namespace,
	}, nil
}

// ListGameServers returns all game servers in the namespace
func (c *Client) ListGameServers(ctx context.Context) (*agonesv1.GameServerList, error) {
	return c.clientset.AgonesV1().GameServers(c.namespace).List(ctx, metav1.ListOptions{})
}

// GetGameServer returns a specific game server
func (c *Client) GetGameServer(ctx context.Context, name string) (*agonesv1.GameServer, error) {
	return c.clientset.AgonesV1().GameServers(c.namespace).Get(ctx, name, metav1.GetOptions{})
}

// CreateGameServer creates a new game server
func (c *Client) CreateGameServer(ctx context.Context, gs *agonesv1.GameServer) (*agonesv1.GameServer, error) {
	return c.clientset.AgonesV1().GameServers(c.namespace).Create(ctx, gs, metav1.CreateOptions{})
}

// DeleteGameServer deletes a game server
func (c *Client) DeleteGameServer(ctx context.Context, name string) error {
	return c.clientset.AgonesV1().GameServers(c.namespace).Delete(ctx, name, metav1.DeleteOptions{})
}

// ListFleets returns all fleets in the namespace
func (c *Client) ListFleets(ctx context.Context) (*agonesv1.FleetList, error) {
	return c.clientset.AgonesV1().Fleets(c.namespace).List(ctx, metav1.ListOptions{})
}

// GetFleet returns a specific fleet
func (c *Client) GetFleet(ctx context.Context, name string) (*agonesv1.Fleet, error) {
	return c.clientset.AgonesV1().Fleets(c.namespace).Get(ctx, name, metav1.GetOptions{})
}

// ScaleFleet scales a fleet to the specified number of replicas
func (c *Client) ScaleFleet(ctx context.Context, fleetName string, replicas int32) error {
	fleet, err := c.GetFleet(ctx, fleetName)
	if err != nil {
		return fmt.Errorf("failed to get fleet: %w", err)
	}

	fleet.Spec.Replicas = replicas
	_, err = c.clientset.AgonesV1().Fleets(c.namespace).Update(ctx, fleet, metav1.UpdateOptions{})
	return err
}

// AllocateGameServer allocates a game server from a fleet
func (c *Client) AllocateGameServer(ctx context.Context, fleetName string) (*allocationv1.GameServerAllocation, error) {
	allocation := &allocationv1.GameServerAllocation{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("%s-allocation-", fleetName),
			Namespace:    c.namespace,
		},
		Spec: allocationv1.GameServerAllocationSpec{
			Required: allocationv1.GameServerSelector{
				LabelSelector: metav1.LabelSelector{
					MatchLabels: map[string]string{
						"agones.dev/fleet": fleetName,
					},
				},
			},
		},
	}

	return c.clientset.AllocationV1().GameServerAllocations(c.namespace).Create(ctx, allocation, metav1.CreateOptions{})
}

// GetFleetAutoscaler returns the autoscaler for a fleet
func (c *Client) GetFleetAutoscaler(ctx context.Context, name string) (*autoscalingv1.FleetAutoscaler, error) {
	return c.clientset.AutoscalingV1().FleetAutoscalers(c.namespace).Get(ctx, name, metav1.GetOptions{})
}

// UpdateFleetAutoscaler updates the autoscaler configuration
func (c *Client) UpdateFleetAutoscaler(ctx context.Context, fas *autoscalingv1.FleetAutoscaler) (*autoscalingv1.FleetAutoscaler, error) {
	return c.clientset.AutoscalingV1().FleetAutoscalers(c.namespace).Update(ctx, fas, metav1.UpdateOptions{})
}

// GetGameServerStatus returns the status of game servers
func (c *Client) GetGameServerStatus(ctx context.Context) (*GameServerStatus, error) {
	gameServers, err := c.ListGameServers(ctx)
	if err != nil {
		return nil, err
	}

	status := &GameServerStatus{
		Total:     len(gameServers.Items),
		Ready:     0,
		Allocated: 0,
		Reserved:  0,
		Shutdown:  0,
		Error:     0,
		GameServers: make([]GameServerInfo, 0),
	}

	for _, gs := range gameServers.Items {
		info := GameServerInfo{
			Name:      gs.Name,
			State:     string(gs.Status.State),
			Address:   gs.Status.Address,
			CreatedAt: gs.CreationTimestamp.Time,
		}

		if len(gs.Status.Ports) > 0 {
			info.Port = gs.Status.Ports[0].Port
		}

		status.GameServers = append(status.GameServers, info)

		switch gs.Status.State {
		case agonesv1.GameServerStateReady:
			status.Ready++
		case agonesv1.GameServerStateAllocated:
			status.Allocated++
		case agonesv1.GameServerStateReserved:
			status.Reserved++
		case agonesv1.GameServerStateShutdown:
			status.Shutdown++
		case agonesv1.GameServerStateError:
			status.Error++
		}
	}

	return status, nil
}

// WatchGameServers watches for changes to game servers
func (c *Client) WatchGameServers(ctx context.Context, callback func(eventType string, gs *agonesv1.GameServer)) error {
	watcher, err := c.clientset.AgonesV1().GameServers(c.namespace).Watch(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}
	defer watcher.Stop()

	for event := range watcher.ResultChan() {
		gs, ok := event.Object.(*agonesv1.GameServer)
		if !ok {
			continue
		}
		callback(string(event.Type), gs)
	}

	return nil
}

// GameServerStatus represents the overall status of game servers
type GameServerStatus struct {
	Total       int
	Ready       int
	Allocated   int
	Reserved    int
	Shutdown    int
	Error       int
	GameServers []GameServerInfo
}

// GameServerInfo represents information about a single game server
type GameServerInfo struct {
	Name      string
	State     string
	Address   string
	Port      int32
	CreatedAt time.Time
}

// CreateSimpleGameServer creates a simple game server for testing
func (c *Client) CreateSimpleGameServer(ctx context.Context, name string) (*agonesv1.GameServer, error) {
	gs := &agonesv1.GameServer{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: name + "-",
			Namespace:    c.namespace,
			Labels: map[string]string{
				"app":         "agis-bot",
				"managed-by":  "agis-bot",
				"environment": "development",
			},
		},
		Spec: agonesv1.GameServerSpec{
			Ports: []agonesv1.GameServerPort{
				{
					Name:          "default",
					PortPolicy:    agonesv1.Dynamic,
					ContainerPort: 7654,
					Protocol:      "UDP",
				},
			},
			Health: agonesv1.Health{
				Disabled:            false,
				PeriodSeconds:       5,
				FailureThreshold:    3,
				InitialDelaySeconds: 5,
			},
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            "game-server",
							Image:           "us-docker.pkg.dev/agones-images/examples/simple-game-server:0.39",
							ImagePullPolicy: corev1.PullIfNotPresent,
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("20m"),
									corev1.ResourceMemory: resource.MustParse("64Mi"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("100m"),
									corev1.ResourceMemory: resource.MustParse("128Mi"),
								},
							},
						},
					},
				},
			},
		},
	}

	return c.CreateGameServer(ctx, gs)
}

// HealthCheck performs a health check on the Agones system
func (c *Client) HealthCheck(ctx context.Context) error {
	// Try to list game servers to verify connectivity
	_, err := c.clientset.AgonesV1().GameServers(c.namespace).List(ctx, metav1.ListOptions{
		Limit: 1,
	})
	
	if err != nil {
		return fmt.Errorf("agones health check failed: %w", err)
	}
	
	log.Printf("Agones client health check passed for namespace: %s", c.namespace)
	return nil
}