package services

import (
	"context"
	"fmt"
	"log"
	"time"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
)

// EnhancedServerService integrates database, Agones, and notifications
type EnhancedServerService struct {
	db            *DatabaseService
	agones        *AgonesService
	notifications *NotificationService
}

// NewEnhancedServerService creates a new enhanced server service
func NewEnhancedServerService(db *DatabaseService, agones *AgonesService, notifications *NotificationService) *EnhancedServerService {
	return &EnhancedServerService{
		db:            db,
		agones:        agones,
		notifications: notifications,
	}
}

// CreateGameServer creates a new game server with full lifecycle management
func (e *EnhancedServerService) CreateGameServer(ctx context.Context, userID, gameType, serverName string, costPerHour int, optionalChannelID ...string) (*GameServer, error) {
	var channelID string
	if len(optionalChannelID) > 0 {
		channelID = optionalChannelID[0]
	}
	// Create database record first
	server := &GameServer{
		DiscordID:      userID,
		Name:           serverName,
		GameType:       gameType,
		Status:         "pending",
		CostPerHour:    costPerHour,
		IsPublic:       false,
		Description:    fmt.Sprintf("A %s server", gameType),
		AgonesStatus:   "Pending",
		LastStatusSync: &time.Time{}, // Initialize with epoch
	}

	err := e.db.SaveGameServer(server)
	if err != nil {
		return nil, fmt.Errorf("failed to save server to database: %v", err)
	}

	// Send initial notification
	e.notifications.NotifyServerStatusChange(ServerStatusUpdate{
		ServerName:     serverName,
		UserID:         userID,
		PreviousStatus: "",
		NewStatus:      "Pending",
		GameType:       gameType,
		ChannelID:      channelID,
	})

	// Start async allocation process
	go e.allocateServerAsync(ctx, server, channelID)

	return server, nil
}

// allocateServerAsync handles the server allocation process asynchronously
func (e *EnhancedServerService) allocateServerAsync(ctx context.Context, server *GameServer, channelID string) {
	log.Printf("Starting allocation for server %s (user: %s)", server.Name, server.DiscordID)

	// Update status to creating
	err := e.db.UpdateServerStatus(server.Name, server.DiscordID, "creating")
	if err != nil {
		log.Printf("Failed to update server status to creating: %v", err)
	} else {
		e.notifications.NotifyServerStatusChange(ServerStatusUpdate{
			ServerName:     server.Name,
			UserID:         server.DiscordID,
			PreviousStatus: "pending",
			NewStatus:      "Creating",
			GameType:       server.GameType,
			ChannelID:      channelID,
		})
	}

	// Allocate GameServer from Agones with retry when capacity is unavailable
	retryInterval := 15 * time.Second
	retryTimeout := 10 * time.Minute
	deadline := time.Now().Add(retryTimeout)
	var agonesInfo *GameServerInfo
	for {
		ai, err := e.agones.AllocateGameServer(ctx, server.GameType, server.Name, server.DiscordID)
		if err == nil {
			agonesInfo = ai
			break
		}
		// First failure: mark as pending and notify, then keep retrying until timeout
		log.Printf("Allocation pending for %s: %v (will retry)", server.Name, err)
		_ = e.db.UpdateServerStatus(server.Name, server.DiscordID, "requested")
		_ = e.notifications.NotifyServerStatusChange(ServerStatusUpdate{
			ServerName:     server.Name,
			UserID:         server.DiscordID,
			PreviousStatus: "creating",
			NewStatus:      "Pending",
			GameType:       server.GameType,
			ChannelID:      channelID,
		})
		if time.Now().After(deadline) {
			log.Printf("Failed to allocate GameServer for %s within timeout", server.Name)
			// Update database with error
			_ = e.db.UpdateServerStatus(server.Name, server.DiscordID, "error")
			_ = e.db.UpdateServerError(server.Name, server.DiscordID, "Allocation timed out waiting for capacity")
			// Notify user of error
			if channelID != "" {
				e.notifications.NotifyServerErrorInChannel(server.DiscordID, server.Name, server.GameType, "Allocation timed out waiting for capacity", channelID)
			} else {
				e.notifications.NotifyServerError(server.DiscordID, server.Name, server.GameType, "Allocation timed out waiting for capacity")
			}
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(retryInterval):
		}
	}

	log.Printf("GameServer allocated: %s (UID: %s)", agonesInfo.Name, agonesInfo.UID)

	// Update database with Kubernetes UID and initial status
	err = e.db.UpdateServerKubernetesInfo(server.Name, server.DiscordID, agonesInfo.UID, string(agonesInfo.Status))
	if err != nil {
		log.Printf("Failed to update server Kubernetes info: %v", err)
	}

	// Start monitoring the GameServer status
	e.monitorGameServerStatus(ctx, server, agonesInfo.UID, channelID)
}

// monitorGameServerStatus monitors a GameServer until it's ready or fails
func (e *EnhancedServerService) monitorGameServerStatus(ctx context.Context, server *GameServer, uid string, channelID string) {
	log.Printf("Starting status monitoring for GameServer %s (UID: %s)", server.Name, uid)

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	timeout := time.After(10 * time.Minute) // 10 minute timeout
	lastNotifiedStatus := ""

	for {
		select {
		case <-ctx.Done():
			log.Printf("Context cancelled for GameServer monitoring: %s", server.Name)
			return
		case <-timeout:
			log.Printf("Timeout reached for GameServer %s", server.Name)
			e.handleServerTimeout(server, channelID)
			return
		case <-ticker.C:
			info, err := e.agones.GetGameServerByUID(ctx, uid)
			if err != nil {
				log.Printf("Error checking GameServer status for %s: %v", server.Name, err)
				continue
			}

			// Update database with latest status
			now := time.Now()
			err = e.db.UpdateServerAgonesStatus(server.Name, server.DiscordID, string(info.Status), &now)
			if err != nil {
				log.Printf("Failed to update Agones status in database: %v", err)
			}

			// Check if we need to notify user of status change
			statusString := e.mapAgonesStatusToUserFriendly(info.Status)
			if statusString != lastNotifiedStatus {
				lastNotifiedStatus = statusString

				update := ServerStatusUpdate{
					ServerName:     server.Name,
					UserID:         server.DiscordID,
					PreviousStatus: e.db.GetServerStatus(server.Name, server.DiscordID),
					NewStatus:      statusString,
					GameType:       server.GameType,
					ChannelID:      channelID,
				}

				// If server is ready, include connection info
				if info.Status == agonesv1.GameServerStateReady || info.Status == agonesv1.GameServerStateAllocated {
					update.Address = info.Address
					update.Port = info.Port

					// Update database with connection info
					e.db.UpdateServerAddress(server.Name, server.DiscordID, info.Address, int(info.Port))
					e.db.UpdateServerStatus(server.Name, server.DiscordID, "ready")

					log.Printf("GameServer %s is ready! Address: %s:%d", server.Name, info.Address, info.Port)

					// Send final notification and stop monitoring
					e.notifications.NotifyServerStatusChange(update)
					return
				} else if info.Status == agonesv1.GameServerStateError || info.Status == agonesv1.GameServerStateUnhealthy {
					// Handle error states
					e.db.UpdateServerStatus(server.Name, server.DiscordID, "error")
					update.ErrorMessage = "GameServer encountered an error during startup"
					e.notifications.NotifyServerStatusChange(update)
					return
				} else {
					// Send intermediate status update
					e.notifications.NotifyServerStatusChange(update)
				}
			}
		}
	}
}

// handleServerTimeout handles when a server takes too long to become ready
func (e *EnhancedServerService) handleServerTimeout(server *GameServer, channelID string) {
	log.Printf("GameServer %s timed out during allocation", server.Name)

	e.db.UpdateServerStatus(server.Name, server.DiscordID, "error")
	e.db.UpdateServerError(server.Name, server.DiscordID, "Server allocation timed out after 10 minutes")

	if channelID != "" {
		e.notifications.NotifyServerErrorInChannel(server.DiscordID, server.Name, server.GameType,
			"Server took too long to start up. Please try again or contact support.", channelID)
	} else {
		e.notifications.NotifyServerError(server.DiscordID, server.Name, server.GameType,
			"Server took too long to start up. Please try again or contact support.")
	}
}

// mapAgonesStatusToUserFriendly converts Agones status to user-friendly status
func (e *EnhancedServerService) mapAgonesStatusToUserFriendly(status agonesv1.GameServerState) string {
	switch status {
	case agonesv1.GameServerStatePortAllocation:
		return "Creating"
	case agonesv1.GameServerStateCreating:
		return "Creating"
	case agonesv1.GameServerStateStarting:
		return "Starting"
	case agonesv1.GameServerStateScheduled:
		return "Starting"
	case agonesv1.GameServerStateRequestReady:
		return "Starting"
	case agonesv1.GameServerStateReady:
		return "Ready"
	case agonesv1.GameServerStateAllocated:
		return "Ready"
	case agonesv1.GameServerStateReserved:
		return "Ready"
	case agonesv1.GameServerStateShutdown:
		return "Shutdown"
	case agonesv1.GameServerStateError:
		return "Error"
	case agonesv1.GameServerStateUnhealthy:
		return "Error"
	default:
		return string(status)
	}
}

// GetEnhancedServerInfo gets server info with live Kubernetes data
func (e *EnhancedServerService) GetEnhancedServerInfo(ctx context.Context, serverName, userID string) (*GameServer, error) {
	// Get database record
	server, err := e.db.GetServerByName(serverName, userID)
	if err != nil {
		return nil, err
	}

	// If we have a Kubernetes UID, get live status
	if server.KubernetesUID != "" {
		info, err := e.agones.GetGameServerByUID(ctx, server.KubernetesUID)
		if err != nil {
			log.Printf("Failed to get live GameServer status for %s: %v", serverName, err)
			// Return database record even if we can't get live status
			return server, nil
		}

		// Update server with live data
		server.AgonesStatus = string(info.Status)
		server.Status = e.mapAgonesStatusToUserFriendly(info.Status)
		if info.Address != "" {
			server.Address = info.Address
			server.Port = int(info.Port)
		}
		now := time.Now()
		server.LastStatusSync = &now

		// Update database with latest info (async to avoid blocking)
		go func() {
			e.db.UpdateServerAgonesStatus(serverName, userID, string(info.Status), &now)
			if info.Address != "" {
				e.db.UpdateServerAddress(serverName, userID, info.Address, int(info.Port))
			}
		}()
	}

	return server, nil
}

// GetUserServersEnhanced gets all user servers with live status
func (e *EnhancedServerService) GetUserServersEnhanced(ctx context.Context, userID string) ([]*GameServer, error) {
	servers, err := e.db.GetUserServers(userID)
	if err != nil {
		return nil, err
	}

	// Update each server with live status if possible
	for _, server := range servers {
		if server.KubernetesUID != "" {
			if info, err := e.agones.GetGameServerByUID(ctx, server.KubernetesUID); err == nil {
				server.AgonesStatus = string(info.Status)
				server.Status = e.mapAgonesStatusToUserFriendly(info.Status)
				if info.Address != "" {
					server.Address = info.Address
					server.Port = int(info.Port)
				}
				now := time.Now()
				server.LastStatusSync = &now
			}
		}
	}

	return servers, nil
}

// DeleteGameServer deletes a game server from both database and Kubernetes
func (e *EnhancedServerService) DeleteGameServer(ctx context.Context, serverName, userID string) error {
	// Get server info
	server, err := e.db.GetServerByName(serverName, userID)
	if err != nil {
		return err
	}

	// Delete from Kubernetes if it exists
	if server.KubernetesUID != "" {
		// First get the actual GameServer info by UID to get the correct Kubernetes name
		gsInfo, err := e.agones.GetGameServerByUID(ctx, server.KubernetesUID)
		if err != nil {
			log.Printf("Failed to get GameServer by UID %s: %v", server.KubernetesUID, err)
			// Continue with database deletion even if we can't find the GameServer
		} else {
			// Delete using the actual Kubernetes GameServer name
			err = e.agones.DeleteGameServer(ctx, gsInfo.Name)
			if err != nil {
				log.Printf("Failed to delete GameServer %s from Kubernetes: %v", gsInfo.Name, err)
				// Continue with database deletion even if Kubernetes deletion fails
			} else {
				log.Printf("Successfully deleted GameServer %s from Kubernetes", gsInfo.Name)
			}
		}
	}

	// Delete from database
	err = e.db.DeleteGameServer(server.ID)
	if err != nil {
		return fmt.Errorf("failed to delete server from database: %v", err)
	}

	// Notify user
	e.notifications.NotifyServerStatusChange(ServerStatusUpdate{
		ServerName:     serverName,
		UserID:         userID,
		PreviousStatus: server.Status,
		NewStatus:      "Deleted",
		GameType:       server.GameType,
	})

	return nil
}
