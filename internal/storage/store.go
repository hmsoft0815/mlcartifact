// Copyright (c) 2026 Michael Lechner. All rights reserved.

// Package storage provides a file-based storage backend for artifacts.
// It manages both the binary content and the associated JSON metadata.
//
// Artifacts can be stored in a global namespace or scoped to a specific user.
// Files are stored with a unique ID prefix to avoid filename collisions and
// to allow retrieval by either the ID or the original filename.
package storage

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

// ArtifactMetadata contains all descriptive information about a stored file.
// It is persisted as a companion .json file alongside the actual artifact data.
type ArtifactMetadata struct {
	ID          string                 `json:"id"`                     // Unique short ID generated at write time
	Filename    string                 `json:"filename"`               // Original filename provided by the client
	VirtualPath string                 `json:"virtual_path,omitempty"` // Hierarchical path (VFS)
	MimeType    string                 `json:"mime_type"`              // Detected or provided MIME type
	Description string                 `json:"description,omitempty"`
	Source      string                 `json:"source,omitempty"`
	UserID      string                 `json:"user_id,omitempty"`     // Optional owner of the artifact
	CreatedAt   time.Time              `json:"created_at"`
	ExpiresAt   time.Time              `json:"expires_at"`            // Scheduled deletion time
	SizeBytes   int64                  `json:"size_bytes,omitempty"`  // Size in bytes
	Metadata    map[string]interface{} `json:"metadata,omitempty"`    // Arbitrary custom metadata
}

// Store handles the persistence of artifacts on the local filesystem.
type Store struct {
	BaseDir string // Root directory where all artifacts and users are stored
	mu      sync.RWMutex
	// Index map[userID]map[virtualPath]artifactID
	index map[string]map[string]string
}

// NewStore initializes a new Store with the given base directory and rebuilds the index.
func NewStore(baseDir string) *Store {
	s := &Store{
		BaseDir: baseDir,
		index:   make(map[string]map[string]string),
	}
	s.rebuildIndex()
	return s
}

// Write saves content and its metadata to the store.
//
// It performs several steps:
// 1. Validates input (UTF-8 checks).
// 2. Determines the storage directory based on the userID (global vs user-scoped).
// 3. Generates a unique ID (based on timestamp and random bytes).
// 4. Detects the MIME type if not provided.
// 5. Writes the binary content and the JSON metadata to disk.
// 6. Updates the in-memory VFS index.
func (s *Store) Write(filename string, content []byte, mimeType string, expiresHours int, source string, userID string, description string, metadata map[string]interface{}, virtualPath string) (*ArtifactMetadata, error) {
	// 1. Validates input (UTF-8 checks).
	if description != "" && !utf8Valid(description) {
		return nil, fmt.Errorf("description contains invalid UTF-8 characters")
	}
	if err := os.MkdirAll(s.BaseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base directory: %w", err)
	}

	prefixDir, err := s.userDir(userID)
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(prefixDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create prefix directory: %w", err)
	}

	// 2. Generate unique ID
	randomID := make([]byte, 4)
	_, _ = rand.Read(randomID)
	id := fmt.Sprintf("%x-%x", time.Now().Unix()%10000, randomID)

	// 3. Paths
	safeFilename := filepath.Base(strings.ReplaceAll(filename, "\\", "/"))
	if safeFilename == "." || safeFilename == "/" || safeFilename == "" {
		safeFilename = "artifact"
	}
	storageName := fmt.Sprintf("%s_%s", id, safeFilename)
	fullPath := filepath.Join(prefixDir, storageName)
	metaPath := fullPath + ".json"

	relFile, err := filepath.Rel(prefixDir, fullPath)
	if err != nil || relFile == "." || relFile == ".." || strings.HasPrefix(relFile, ".."+string(filepath.Separator)) {
		return nil, ErrPathEscape
	}

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
	vPath := NormalizePath(virtualPath)
	if virtualPath == "" {
		vPath = "" // Don't index empty paths
	}

	meta := &ArtifactMetadata{
		ID:          id,
		Filename:    safeFilename,
		VirtualPath: vPath,
		MimeType:    mimeType,
		Description: description,
		Source:      source,
		UserID:      userID,
		CreatedAt:   time.Now(),
		ExpiresAt:   expiresAt,
		SizeBytes:   int64(len(content)),
		Metadata:    metadata,
	}

	metaBytes, _ := json.MarshalIndent(meta, "", "  ")
	if err := os.WriteFile(metaPath, metaBytes, 0644); err != nil {
		return nil, fmt.Errorf("failed to write metadata: %w", err)
	}

	// 8. Update index
	if vPath != "" {
		s.mu.Lock()
		uID := userID
		if uID == "" {
			uID = "global"
		}
		if s.index[uID] == nil {
			s.index[uID] = make(map[string]string)
		}
		s.index[uID][vPath] = id
		s.mu.Unlock()
	}

	return meta, nil
}

// Read retrieves content and metadata for a given ID, filename, or virtual path.
// If multiple files match a filename, the newest one (highest ID prefix) is returned.
func (s *Store) Read(idOrPath string, userID string) ([]byte, *ArtifactMetadata, error) {
	if err := ValidateUserID(userID); err != nil {
		return nil, nil, err
	}

	uID := userID
	if uID == "" {
		uID = "global"
	}

	lookupID := idOrPath

	// 1. Try VFS index if it looks like a path
	if strings.HasPrefix(idOrPath, "/") {
		s.mu.RLock()
		if userIdx, ok := s.index[uID]; ok {
			p := NormalizePath(idOrPath)
			if id, exists := userIdx[p]; exists {
				lookupID = id
			}
		}
		s.mu.RUnlock()
	}

	prefixDir, err := s.userDir(userID)
	if err != nil {
		return nil, nil, err
	}

	// Find the file. ID is prefix of storage name: {id}_{filename}
	files, err := os.ReadDir(prefixDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, ErrArtifactNotFound
		}
		return nil, nil, fmt.Errorf("failed to read storage directory: %w", err)
	}

	for _, f := range files {
		// CRITICAL: We must NOT match the metadata file here when looking for content.
		// Artifact files are named {id}_{filename}.
		// Metadata files are named {id}_{filename}.json.
		if f.IsDir() || strings.HasSuffix(f.Name(), ".json.json") {
			continue
		}

		if strings.HasSuffix(f.Name(), ".json") {
			// Check if this is a metadata file by looking for the artifact file
			artifactName := strings.TrimSuffix(f.Name(), ".json")
			if _, err := os.Stat(filepath.Join(prefixDir, artifactName)); err == nil {
				// Artifact file exists, so THIS file is definitely metadata
				continue
			}
		}

		// Storage format is: {id}_{filename}
		parts := strings.SplitN(f.Name(), "_", 2)
		if len(parts) != 2 {
			continue
		}

		isMatch := (parts[0] == lookupID) || (parts[1] == lookupID)
		if isMatch {
			fullPath := filepath.Join(prefixDir, f.Name())
			metaPath := fullPath + ".json"

			data, err := os.ReadFile(fullPath)
			if err != nil {
				if os.IsNotExist(err) {
					return nil, nil, ErrArtifactNotFound
				}
				return nil, nil, fmt.Errorf("failed to read artifact data: %w", err)
			}

			metaData, err := os.ReadFile(metaPath)
			if err != nil {
				if os.IsNotExist(err) {
					return nil, nil, ErrArtifactNotFound
				}
				return nil, nil, fmt.Errorf("failed to read artifact metadata: %w", err)
			}

			var meta ArtifactMetadata
			if err := json.Unmarshal(metaData, &meta); err != nil {
				return nil, nil, fmt.Errorf("failed to parse artifact metadata: %w", err)
			}
			if meta.SizeBytes == 0 {
				meta.SizeBytes = int64(len(data))
			}

			return data, &meta, nil
		}
	}

	return nil, nil, ErrArtifactNotFound
}

// List returns artifacts for a specific user.
// If dirPath is empty, it returns a flat list of all artifacts.
// If dirPath is set, it returns items (files and virtual folders) in that virtual directory.
func (s *Store) List(userID string, limit, offset int, dirPath string) ([]*ArtifactMetadata, error) {
	if err := ValidateUserID(userID); err != nil {
		return nil, err
	}

	if dirPath != "" {
		return s.ListVFS(userID, dirPath, limit, offset)
	}

	prefixDir, err := s.userDir(userID)
	if err != nil {
		return nil, err
	}

	files, err := os.ReadDir(prefixDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []*ArtifactMetadata{}, nil
		}
		return nil, fmt.Errorf("failed to read storage directory: %w", err)
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
				if meta.SizeBytes == 0 {
					dataPath := filepath.Join(prefixDir, strings.TrimSuffix(f.Name(), ".json"))
					if fi, err := os.Stat(dataPath); err == nil {
						meta.SizeBytes = fi.Size()
					}
				}
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

// Patch modifies an existing artifact's content.
// Patch only works for text files; it directly returns ErrBinaryFile if called for a binary file.
func (s *Store) Patch(idOrPath string, userID string, patchContent []byte, lineStart, lineEnd int, shouldAppend bool) (int64, error) {
	if err := ValidateUserID(userID); err != nil {
		return 0, err
	}

	oldContent, meta, err := s.Read(idOrPath, userID)
	if err != nil {
		return 0, err
	}

	// Patch is only supported for text files. Directly return if the artifact or patch content is binary.
	if IsBinary(meta.Filename, meta.MimeType, oldContent) {
		return 0, ErrBinaryFile
	}
	if bytes.IndexByte(patchContent, 0) != -1 || !utf8.Valid(patchContent) {
		return 0, ErrBinaryFile
	}

	var newContent []byte
	if shouldAppend {
		newContent = append(oldContent, patchContent...)
	} else {
		// Line-based patching
		lines := strings.Split(string(oldContent), "\n")
		patchLines := strings.Split(string(patchContent), "\n")

		if lineStart < 0 {
			lineStart = 0
		}
		if lineStart > len(lines) {
			lineStart = len(lines)
		}
		if lineEnd < lineStart {
			lineEnd = lineStart
		}
		if lineEnd > len(lines) {
			lineEnd = len(lines)
		}

		// Construct new lines
		resultLines := append([]string{}, lines[:lineStart]...)
		resultLines = append(resultLines, patchLines...)
		if lineEnd < len(lines) {
			resultLines = append(resultLines, lines[lineEnd:]...)
		}

		// Remove empty trailing string if initial file was empty and we added content
		if len(lines) == 1 && lines[0] == "" && len(patchLines) > 0 {
			// This is a special case for patching empty files
			newContent = patchContent
		} else {
			newContent = []byte(strings.Join(resultLines, "\n"))
		}
	}

	// Overwrite existing file
	prefixDir, err := s.userDir(userID)
	if err != nil {
		return 0, err
	}

	// We need to find the actual file on disk (ID_Filename)
	storageName := fmt.Sprintf("%s_%s", meta.ID, meta.Filename)
	fullPath := filepath.Join(prefixDir, storageName)

	relFile, err := filepath.Rel(prefixDir, fullPath)
	if err != nil || relFile == "." || relFile == ".." || strings.HasPrefix(relFile, ".."+string(filepath.Separator)) {
		return 0, ErrPathEscape
	}

	if err := os.WriteFile(fullPath, newContent, 0644); err != nil {
		return 0, fmt.Errorf("failed to update data: %w", err)
	}

	// Update metadata with new size
	meta.SizeBytes = int64(len(newContent))
	metaPath := fullPath + ".json"
	if metaBytes, err := json.MarshalIndent(meta, "", "  "); err == nil {
		_ = os.WriteFile(metaPath, metaBytes, 0644)
	}

	return int64(len(newContent)), nil
}

// Delete removes an artifact and its associated metadata JSON file.
// Returns true if the artifact was found and deleted, false otherwise.
func (s *Store) Delete(idOrPath string, userID string) (bool, error) {
	if err := ValidateUserID(userID); err != nil {
		return false, err
	}

	uID := userID
	if uID == "" {
		uID = "global"
	}

	lookupID := idOrPath
	var vPath string

	// 1. Try VFS index if it looks like a path
	if strings.HasPrefix(idOrPath, "/") {
		s.mu.RLock()
		if userIdx, ok := s.index[uID]; ok {
			p := NormalizePath(idOrPath)
			if id, exists := userIdx[p]; exists {
				lookupID = id
				vPath = p
			}
		}
		s.mu.RUnlock()
	}

	prefixDir, err := s.userDir(userID)
	if err != nil {
		return false, err
	}

	files, err := os.ReadDir(prefixDir)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to read storage directory: %w", err)
	}

	for _, f := range files {
		if f.IsDir() || strings.HasSuffix(f.Name(), ".json.json") {
			continue
		}
		if strings.HasSuffix(f.Name(), ".json") {
			artifactName := strings.TrimSuffix(f.Name(), ".json")
			if _, err := os.Stat(filepath.Join(prefixDir, artifactName)); err == nil {
				continue
			}
		}

		parts := strings.SplitN(f.Name(), "_", 2)
		if len(parts) != 2 {
			continue
		}

		isMatch := (parts[0] == lookupID) || (parts[1] == lookupID)
		if isMatch {
			fullPath := filepath.Join(prefixDir, f.Name())
			metaPath := fullPath + ".json"

			// If we didn't have the path yet (looked up by ID), try to find it in meta
			if vPath == "" {
				if metaData, err := os.ReadFile(metaPath); err == nil {
					var meta ArtifactMetadata
					if err := json.Unmarshal(metaData, &meta); err == nil {
						vPath = meta.VirtualPath
					}
				}
			}

			_ = os.Remove(fullPath)
			_ = os.Remove(metaPath)

			// Update index if we found a path
			if vPath != "" {
				s.mu.Lock()
				if userIdx, ok := s.index[uID]; ok {
					delete(userIdx, vPath)
				}
				s.mu.Unlock()
			}

			return true, nil
		}
	}

	return false, nil
}

// Cleanup performs a recursive walk of the BaseDir and removes any artifacts
// whose expiration time (ExpiresAt) has passed.
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
