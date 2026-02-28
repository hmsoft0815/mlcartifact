// Copyright (c) 2026 Michael Lechner. All rights reserved.
package storage

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"
)

func utf8Valid(s string) bool {
	return utf8.ValidString(s)
}

type ArtifactMetadata struct {
	ID          string                 `json:"id"`
	Filename    string                 `json:"filename"`
	MimeType    string                 `json:"mime_type"`
	Description string                 `json:"description,omitempty"`
	Source      string                 `json:"source,omitempty"`
	UserID      string                 `json:"user_id,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	ExpiresAt   time.Time              `json:"expires_at"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

type Store struct {
	BaseDir string
}

func NewStore(baseDir string) *Store {
	return &Store{BaseDir: baseDir}
}

// Write saves content to the store.
func (s *Store) Write(filename string, content []byte, mimeType string, expiresHours int, source string, userID string, description string, metadata map[string]interface{}) (*ArtifactMetadata, error) {
	if description != "" && utf8Valid(description) == false {
		return nil, fmt.Errorf("description contains invalid UTF-8 characters")
	}
	if err := os.MkdirAll(s.BaseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base directory: %w", err)
	}

	// 1. Determine storage prefix (global vs user)
	prefixDir := filepath.Join(s.BaseDir, "global")
	if userID != "" {
		prefixDir = filepath.Join(s.BaseDir, "users", userID)
	}

	if err := os.MkdirAll(prefixDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create prefix directory: %w", err)
	}

	// 2. Generate unique ID
	randomID := make([]byte, 4)
	_, _ = rand.Read(randomID)
	id := fmt.Sprintf("%x-%x", time.Now().Unix()%10000, randomID)

	// 3. Paths
	safeFilename := filepath.Base(filename)
	storageName := fmt.Sprintf("%s_%s", id, safeFilename)
	fullPath := filepath.Join(prefixDir, storageName)
	metaPath := fullPath + ".json"

	// 4. Expiration
	if expiresHours <= 0 {
		expiresHours = 24
	}
	expiresAt := time.Now().Add(time.Duration(expiresHours) * time.Hour)

	// 5. Detect MIME if missing
	if mimeType == "" {
		mimeType = DetectMimeType(filename)
	}

	// 6. Write file
	if err := os.WriteFile(fullPath, content, 0644); err != nil {
		return nil, fmt.Errorf("failed to write data: %w", err)
	}

	// 7. Write metadata
	meta := &ArtifactMetadata{
		ID:          id,
		Filename:    safeFilename,
		MimeType:    mimeType,
		Description: description,
		Source:      source,
		UserID:      userID,
		CreatedAt:   time.Now(),
		ExpiresAt:   expiresAt,
		Metadata:    metadata,
	}

	metaBytes, _ := json.MarshalIndent(meta, "", "  ")
	if err := os.WriteFile(metaPath, metaBytes, 0644); err != nil {
		return nil, fmt.Errorf("failed to write metadata: %w", err)
	}

	return meta, nil
}

// Read retrieves content and metadata.
func (s *Store) Read(idOrFilename string, userID string) ([]byte, *ArtifactMetadata, error) {
	prefixDir := filepath.Join(s.BaseDir, "global")
	if userID != "" {
		prefixDir = filepath.Join(s.BaseDir, "users", userID)
	}

	// Find the file. ID is prefix of storage name: {id}_{filename}
	files, err := os.ReadDir(prefixDir)
	if err != nil {
		return nil, nil, err
	}

	for _, f := range files {
		// Matches if ID is prefix OR filename is suffix (following the ID_ separator)
		isMatch := strings.HasPrefix(f.Name(), idOrFilename) || strings.HasSuffix(f.Name(), "_"+idOrFilename)
		if !f.IsDir() && isMatch && !strings.HasSuffix(f.Name(), ".json") {
			fullPath := filepath.Join(prefixDir, f.Name())
			metaPath := fullPath + ".json"

			data, err := os.ReadFile(fullPath)
			if err != nil {
				return nil, nil, err
			}

			metaData, err := os.ReadFile(metaPath)
			if err != nil {
				return nil, nil, err
			}

			var meta ArtifactMetadata
			if err := json.Unmarshal(metaData, &meta); err != nil {
				return nil, nil, err
			}

			return data, &meta, nil
		}
	}

	return nil, nil, fmt.Errorf("artifact not found")
}

// List returns all artifacts for a user (or global) with pagination.
func (s *Store) List(userID string, limit, offset int) ([]*ArtifactMetadata, error) {
	prefixDir := filepath.Join(s.BaseDir, "global")
	if userID != "" {
		prefixDir = filepath.Join(s.BaseDir, "users", userID)
	}

	files, err := os.ReadDir(prefixDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []*ArtifactMetadata{}, nil
		}
		return nil, err
	}

	var results []*ArtifactMetadata
	for _, f := range files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".json") {
			metaData, err := os.ReadFile(filepath.Join(prefixDir, f.Name()))
			if err != nil {
				continue
			}

			var meta ArtifactMetadata
			if err := json.Unmarshal(metaData, &meta); err == nil {
				results = append(results, &meta)
			}
		}
	}

	// Pagination
	if offset > len(results) {
		return []*ArtifactMetadata{}, nil
	}
	end := len(results)
	if limit > 0 {
		end = offset + limit
		if end > len(results) {
			end = len(results)
		}
	}

	return results[offset:end], nil
}

// Delete removes an artifact and its metadata.
func (s *Store) Delete(idOrFilename string, userID string) (bool, error) {
	prefixDir := filepath.Join(s.BaseDir, "global")
	if userID != "" {
		prefixDir = filepath.Join(s.BaseDir, "users", userID)
	}

	files, err := os.ReadDir(prefixDir)
	if err != nil {
		return false, err
	}

	for _, f := range files {
		isMatch := strings.HasPrefix(f.Name(), idOrFilename) || strings.HasSuffix(f.Name(), "_"+idOrFilename)
		if !f.IsDir() && isMatch && !strings.HasSuffix(f.Name(), ".json") {
			fullPath := filepath.Join(prefixDir, f.Name())
			metaPath := fullPath + ".json"

			_ = os.Remove(fullPath)
			_ = os.Remove(metaPath)
			return true, nil
		}
	}

	return false, nil
}

// Cleanup removes expired artifacts recursively.
func (s *Store) Cleanup() {
	_ = filepath.Walk(s.BaseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".json") {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		var meta ArtifactMetadata
		if err := json.Unmarshal(data, &meta); err == nil {
			if time.Now().After(meta.ExpiresAt) {
				_ = os.Remove(strings.TrimSuffix(path, ".json"))
				_ = os.Remove(path)
			}
		}
		return nil
	})
}

func DetectMimeType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".md":
		return "text/markdown"
	case ".html", ".htm":
		return "text/html"
	case ".json":
		return "application/json"
	case ".csv":
		return "text/csv"
	case ".txt", ".log":
		return "text/plain"
	case ".js":
		return "application/javascript"
	case ".ts":
		return "application/x-typescript"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".svg":
		return "image/svg+xml"
	default:
		return "application/octet-stream"
	}
}
