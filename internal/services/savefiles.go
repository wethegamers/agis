package services

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

// SaveFileService handles exporting and managing save files for game servers
type SaveFileService struct {
	basePath string
}

// SaveFileExport represents a save file export
type SaveFileExport struct {
	ServerID    int
	ServerName  string
	GameType    string
	FilePath    string
	DownloadURL string
	ExportedAt  time.Time
	ExpiresAt   time.Time
	FileSize    int64
}

// NewSaveFileService creates a new save file service
func NewSaveFileService(basePath string) *SaveFileService {
	if basePath == "" {
		// Try multiple writable locations in order of preference
		tryPaths := []string{
			"/app/exports",        // Preferred: mounted writable volume
			"/tmp/agis-saves",     // Alternative: tmp directory
			"/app/saves",          // Alternative: app directory
			"/data/agis-saves",    // Alternative: data volume mount
			"/var/tmp/agis-saves", // Alternative: var tmp
			"./saves",             // Fallback: relative to current directory
		}

		for _, path := range tryPaths {
			if err := os.MkdirAll(path, 0755); err == nil {
				basePath = path
				log.Printf("SaveFileService: Using writable path: %s", basePath)
				break
			} else {
				log.Printf("SaveFileService: Failed to create path %s: %v", path, err)
			}
		}

		// Final fallback if nothing worked
		if basePath == "" {
			basePath = "./saves"
			log.Printf("SaveFileService: Using fallback path: %s", basePath)
		}
	}

	// Ensure the base directory exists
	if err := os.MkdirAll(basePath, 0755); err != nil {
		log.Printf("⚠️ Failed to create save file directory: %v", err)
		// Try to create in current directory as final fallback
		if fallbackErr := os.MkdirAll("./saves", 0755); fallbackErr == nil {
			basePath = "./saves"
			log.Printf("📁 Using fallback save directory: %s", basePath)
		}
	} else {
		log.Printf("📁 Save file directory ready: %s", basePath)
	}

	return &SaveFileService{
		basePath: basePath,
	}
}

// ExportServerSave exports a server's save files
func (s *SaveFileService) ExportServerSave(server *GameServer) (*SaveFileExport, error) {
	if server == nil {
		return nil, fmt.Errorf("server cannot be nil")
	}

	// Create a directory for this export
	exportDir := filepath.Join(s.basePath, fmt.Sprintf("%s_%d_%d", server.Name, server.ID, time.Now().Unix()))
	if err := os.MkdirAll(exportDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create export directory: %v", err)
	}

	var archivePath string
	var err error

	// Export save based on game type
	switch server.GameType {
	case "minecraft":
		archivePath, err = s.exportMinecraftSave(server, exportDir)
	case "terraria":
		archivePath, err = s.exportTerrariaSave(server, exportDir)
	case "cs2":
		archivePath, err = s.exportCS2Save(server, exportDir)
	case "gmod":
		archivePath, err = s.exportGModSave(server, exportDir)
	default:
		return nil, fmt.Errorf("unsupported game type: %s", server.GameType)
	}

	if err != nil {
		// Clean up on error
		os.RemoveAll(exportDir)
		return nil, fmt.Errorf("failed to export %s save: %v", server.GameType, err)
	}

	// Get file info
	fileInfo, err := os.Stat(archivePath)
	if err != nil {
		os.RemoveAll(exportDir)
		return nil, fmt.Errorf("failed to get file info: %v", err)
	}

	// Create export record
	export := &SaveFileExport{
		ServerID:    server.ID,
		ServerName:  server.Name,
		GameType:    server.GameType,
		FilePath:    archivePath,
		DownloadURL: fmt.Sprintf("/api/saves/download/%d", server.ID), // This would be handled by web API
		ExportedAt:  time.Now(),
		ExpiresAt:   time.Now().Add(24 * time.Hour), // Expire after 24 hours
		FileSize:    fileInfo.Size(),
	}

	log.Printf("📦 Exported save file for %s server '%s' (%d bytes)", server.GameType, server.Name, fileInfo.Size())
	return export, nil
}

// exportMinecraftSave exports Minecraft world data
func (s *SaveFileService) exportMinecraftSave(server *GameServer, exportDir string) (string, error) {
	// In a real implementation, this would:
	// 1. Connect to the Kubernetes pod
	// 2. Copy the world data from /data/world
	// 3. Create a tar.gz archive
	// 4. Return the path to the archive

	// For now, create a placeholder file
	archivePath := filepath.Join(exportDir, fmt.Sprintf("%s_world.tar.gz", server.Name))
	placeholderContent := fmt.Sprintf("Minecraft world save for %s\nExported at: %s\n", server.Name, time.Now().Format(time.RFC3339))

	err := os.WriteFile(archivePath, []byte(placeholderContent), 0644)
	if err != nil {
		return "", err
	}

	return archivePath, nil
}

// exportTerrariaSave exports Terraria world data
func (s *SaveFileService) exportTerrariaSave(server *GameServer, exportDir string) (string, error) {
	// Similar to Minecraft but for Terraria world files (.wld)
	archivePath := filepath.Join(exportDir, fmt.Sprintf("%s_world.zip", server.Name))
	placeholderContent := fmt.Sprintf("Terraria world save for %s\nExported at: %s\n", server.Name, time.Now().Format(time.RFC3339))

	err := os.WriteFile(archivePath, []byte(placeholderContent), 0644)
	if err != nil {
		return "", err
	}

	return archivePath, nil
}

// exportCS2Save exports CS2 server configuration and custom maps
func (s *SaveFileService) exportCS2Save(server *GameServer, exportDir string) (string, error) {
	// CS2 saves would include custom maps, server configs, etc.
	archivePath := filepath.Join(exportDir, fmt.Sprintf("%s_config.tar.gz", server.Name))
	placeholderContent := fmt.Sprintf("CS2 server configuration for %s\nExported at: %s\n", server.Name, time.Now().Format(time.RFC3339))

	err := os.WriteFile(archivePath, []byte(placeholderContent), 0644)
	if err != nil {
		return "", err
	}

	return archivePath, nil
}

// exportGModSave exports Garry's Mod server data
func (s *SaveFileService) exportGModSave(server *GameServer, exportDir string) (string, error) {
	// GMod saves would include addons, saves, server configs
	archivePath := filepath.Join(exportDir, fmt.Sprintf("%s_data.tar.gz", server.Name))
	placeholderContent := fmt.Sprintf("Garry's Mod server data for %s\nExported at: %s\n", server.Name, time.Now().Format(time.RFC3339))

	err := os.WriteFile(archivePath, []byte(placeholderContent), 0644)
	if err != nil {
		return "", err
	}

	return archivePath, nil
}

// CleanupExpiredExports removes expired save file exports
func (s *SaveFileService) CleanupExpiredExports() error {
	entries, err := os.ReadDir(s.basePath)
	if err != nil {
		return fmt.Errorf("failed to read save directory: %v", err)
	}

	now := time.Now()
	cleaned := 0

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		dirPath := filepath.Join(s.basePath, entry.Name())
		info, err := entry.Info()
		if err != nil {
			continue
		}

		// Remove directories older than 24 hours
		if now.Sub(info.ModTime()) > 24*time.Hour {
			if err := os.RemoveAll(dirPath); err != nil {
				log.Printf("⚠️ Failed to remove expired save export %s: %v", entry.Name(), err)
			} else {
				cleaned++
			}
		}
	}

	if cleaned > 0 {
		log.Printf("🧹 Cleaned up %d expired save exports", cleaned)
	}

	return nil
}

// GetExportSize returns the total size of all exports
func (s *SaveFileService) GetExportSize() (int64, error) {
	var totalSize int64

	err := filepath.Walk(s.basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Continue walking
		}
		if !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})

	return totalSize, err
}

// CopyFile copies a file from src to dst
func (s *SaveFileService) CopyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}
