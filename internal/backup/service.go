package backup

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// BackupService handles server backup and restore operations
type BackupService struct {
	minioClient  *minio.Client
	bucketName   string
	encryptionKey []byte
	enabled      bool
}

// ServerBackup represents a server backup
type ServerBackup struct {
	ID           string                 `json:"id"`
	ServerID     int                    `json:"server_id"`
	ServerName   string                 `json:"server_name"`
	DiscordID    string                 `json:"discord_id"`
	GameType     string                 `json:"game_type"`
	Config       map[string]interface{} `json:"config"`
	WorldData    []byte                 `json:"world_data,omitempty"`
	PlayerData   []byte                 `json:"player_data,omitempty"`
	Plugins      []string               `json:"plugins,omitempty"`
	Mods         []string               `json:"mods,omitempty"`
	Size         int64                  `json:"size"`
	CreatedAt    time.Time              `json:"created_at"`
	ExpiresAt    time.Time              `json:"expires_at"`
	Compressed   bool                   `json:"compressed"`
	Encrypted    bool                   `json:"encrypted"`
}

// NewBackupService creates a new backup service
func NewBackupService(endpoint, accessKey, secretKey, bucketName string, useSSL bool, encryptionKey string) (*BackupService, error) {
	// If no endpoint, service is disabled
	if endpoint == "" {
		log.Println("ℹ️ Backup service disabled (no S3 endpoint configured)")
		return &BackupService{enabled: false}, nil
	}

	// Initialize Minio client (S3-compatible)
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create S3 client: %v", err)
	}

	// Verify bucket exists
	ctx := context.Background()
	exists, err := minioClient.BucketExists(ctx, bucketName)
	if err != nil {
		return nil, fmt.Errorf("failed to check bucket: %v", err)
	}

	if !exists {
		// Create bucket if it doesn't exist
		err = minioClient.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to create bucket: %v", err)
		}
		log.Printf("✅ Created S3 bucket: %s", bucketName)
	}

	// Derive encryption key from passphrase
	key := sha256.Sum256([]byte(encryptionKey))

	return &BackupService{
		minioClient:   minioClient,
		bucketName:    bucketName,
		encryptionKey: key[:],
		enabled:       true,
	}, nil
}

// CreateBackup creates a backup of a server
func (s *BackupService) CreateBackup(ctx context.Context, backup *ServerBackup) error {
	if !s.enabled {
		return fmt.Errorf("backup service is disabled")
	}

	// Generate backup ID
	backup.ID = fmt.Sprintf("%s-%s-%d", backup.DiscordID, backup.ServerName, time.Now().Unix())
	backup.CreatedAt = time.Now()
	backup.ExpiresAt = time.Now().AddDate(0, 0, 30) // 30 days expiration

	// Serialize backup to JSON
	data, err := json.Marshal(backup)
	if err != nil {
		return fmt.Errorf("failed to serialize backup: %v", err)
	}

	// Compress data
	var compressed bytes.Buffer
	gzipWriter := gzip.NewWriter(&compressed)
	if _, err := gzipWriter.Write(data); err != nil {
		return fmt.Errorf("failed to compress backup: %v", err)
	}
	if err := gzipWriter.Close(); err != nil {
		return fmt.Errorf("failed to close gzip writer: %v", err)
	}
	backup.Compressed = true

	// Encrypt data
	encrypted, err := s.encrypt(compressed.Bytes())
	if err != nil {
		return fmt.Errorf("failed to encrypt backup: %v", err)
	}
	backup.Encrypted = true
	backup.Size = int64(len(encrypted))

	// Upload to S3
	objectName := fmt.Sprintf("backups/%s/%s.backup", backup.DiscordID, backup.ID)
	_, err = s.minioClient.PutObject(ctx, s.bucketName, objectName, bytes.NewReader(encrypted), int64(len(encrypted)), minio.PutObjectOptions{
		ContentType: "application/octet-stream",
		UserMetadata: map[string]string{
			"server-id":   fmt.Sprintf("%d", backup.ServerID),
			"server-name": backup.ServerName,
			"game-type":   backup.GameType,
			"discord-id":  backup.DiscordID,
			"expires-at":  backup.ExpiresAt.Format(time.RFC3339),
		},
	})

	if err != nil {
		return fmt.Errorf("failed to upload backup: %v", err)
	}

	log.Printf("✅ Created backup: %s (size: %d bytes, compressed: %v, encrypted: %v)",
		backup.ID, backup.Size, backup.Compressed, backup.Encrypted)

	return nil
}

// RestoreBackup restores a server from backup
func (s *BackupService) RestoreBackup(ctx context.Context, backupID, discordID string) (*ServerBackup, error) {
	if !s.enabled {
		return nil, fmt.Errorf("backup service is disabled")
	}

	// Download from S3
	objectName := fmt.Sprintf("backups/%s/%s.backup", discordID, backupID)
	object, err := s.minioClient.GetObject(ctx, s.bucketName, objectName, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to download backup: %v", err)
	}
	defer object.Close()

	// Read encrypted data
	encrypted, err := io.ReadAll(object)
	if err != nil {
		return nil, fmt.Errorf("failed to read backup: %v", err)
	}

	// Decrypt data
	decrypted, err := s.decrypt(encrypted)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt backup: %v", err)
	}

	// Decompress data
	gzipReader, err := gzip.NewReader(bytes.NewReader(decrypted))
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %v", err)
	}
	defer gzipReader.Close()

	decompressed, err := io.ReadAll(gzipReader)
	if err != nil {
		return nil, fmt.Errorf("failed to decompress backup: %v", err)
	}

	// Deserialize backup
	var backup ServerBackup
	if err := json.Unmarshal(decompressed, &backup); err != nil {
		return nil, fmt.Errorf("failed to deserialize backup: %v", err)
	}

	log.Printf("✅ Restored backup: %s", backupID)
	return &backup, nil
}

// ListBackups lists all backups for a user
func (s *BackupService) ListBackups(ctx context.Context, discordID string) ([]ServerBackup, error) {
	if !s.enabled {
		return nil, fmt.Errorf("backup service is disabled")
	}

	prefix := fmt.Sprintf("backups/%s/", discordID)
	objectCh := s.minioClient.ListObjects(ctx, s.bucketName, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	})

	var backups []ServerBackup
	for object := range objectCh {
		if object.Err != nil {
			log.Printf("Error listing backups: %v", object.Err)
			continue
		}

		// Parse metadata to create backup summary
		backup := ServerBackup{
			ID:         object.Key[len(prefix) : len(object.Key)-7], // Remove prefix and .backup
			Size:       object.Size,
			CreatedAt:  object.LastModified,
			Compressed: true,
			Encrypted:  true,
		}

		// Extract metadata if available
		if serverName, ok := object.UserMetadata["Server-Name"]; ok {
			backup.ServerName = serverName
		}
		if gameType, ok := object.UserMetadata["Game-Type"]; ok {
			backup.GameType = gameType
		}

		backups = append(backups, backup)
	}

	return backups, nil
}

// DeleteBackup deletes a backup
func (s *BackupService) DeleteBackup(ctx context.Context, backupID, discordID string) error {
	if !s.enabled {
		return fmt.Errorf("backup service is disabled")
	}

	objectName := fmt.Sprintf("backups/%s/%s.backup", discordID, backupID)
	err := s.minioClient.RemoveObject(ctx, s.bucketName, objectName, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete backup: %v", err)
	}

	log.Printf("🗑️ Deleted backup: %s", backupID)
	return nil
}

// CleanupExpiredBackups removes expired backups
func (s *BackupService) CleanupExpiredBackups(ctx context.Context) error {
	if !s.enabled {
		return nil
	}

	objectCh := s.minioClient.ListObjects(ctx, s.bucketName, minio.ListObjectsOptions{
		Prefix:    "backups/",
		Recursive: true,
	})

	now := time.Now()
	deleted := 0

	for object := range objectCh {
		if object.Err != nil {
			log.Printf("Error during cleanup: %v", object.Err)
			continue
		}

		// Check if expired (older than 30 days)
		if now.Sub(object.LastModified) > 30*24*time.Hour {
			err := s.minioClient.RemoveObject(ctx, s.bucketName, object.Key, minio.RemoveObjectOptions{})
			if err != nil {
				log.Printf("Failed to delete expired backup %s: %v", object.Key, err)
				continue
			}
			deleted++
		}
	}

	if deleted > 0 {
		log.Printf("🧹 Cleaned up %d expired backups", deleted)
	}

	return nil
}

// encrypt encrypts data using AES-256-GCM
func (s *BackupService) encrypt(data []byte) ([]byte, error) {
	block, err := aes.NewCipher(s.encryptionKey)
	if err != nil {
		return nil, err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// Generate nonce
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	// Encrypt and append nonce
	ciphertext := aesGCM.Seal(nonce, nonce, data, nil)
	return ciphertext, nil
}

// decrypt decrypts data using AES-256-GCM
func (s *BackupService) decrypt(data []byte) ([]byte, error) {
	block, err := aes.NewCipher(s.encryptionKey)
	if err != nil {
		return nil, err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := aesGCM.NonceSize()
	if len(data) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	// Extract nonce and ciphertext
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]

	// Decrypt
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

// IsEnabled returns whether backup service is enabled
func (s *BackupService) IsEnabled() bool {
	return s.enabled
}
