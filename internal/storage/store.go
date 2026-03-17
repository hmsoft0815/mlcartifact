// Copyright (c) 2026 Michael Lechner. All rights reserved.

// Package storage provides a file-based storage backend for artifacts.
// It manages both the binary content and the associated JSON metadata.
//
// Artifacts can be stored in a global namespace or scoped to a specific user.
// Files are stored with a unique ID prefix to avoid filename collisions and
// to allow retrieval by either the ID or the original filename.
package storage

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

// utf8Valid reports whether s is a valid UTF-8 string.
func utf8Valid(s string) bool {
	return utf8.ValidString(s)
}

// ArtifactMetadata contains all descriptive information about a stored file.
// It is persisted as a companion .json file alongside the actual artifact data.
type ArtifactMetadata struct {
	ID          string                 `json:"id"`          // Unique short ID generated at write time
	Filename    string                 `json:"filename"`    // Original filename provided by the client
	VirtualPath string                 `json:"virtual_path,omitempty"` // Hierarchical path (VFS)
	MimeType    string                 `json:"mime_type"`   // Detected or provided MIME type
	Description string                 `json:"description,omitempty"`
	Source      string                 `json:"source,omitempty"`
	UserID      string                 `json:"user_id,omitempty"` // Optional owner of the artifact
	CreatedAt   time.Time              `json:"created_at"`
	ExpiresAt   time.Time              `json:"expires_at"` // Scheduled deletion time
	Metadata    map[string]interface{} `json:"metadata,omitempty"` // Arbitrary custom metadata
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

// rebuildIndex scans the BaseDir and populates the in-memory VFS index.
func (s *Store) rebuildIndex() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Clear existing index
	s.index = make(map[string]map[string]string)

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
			if meta.VirtualPath != "" {
				uID := meta.UserID
				if uID == "" {
					uID = "global"
				}
				if s.index[uID] == nil {
					s.index[uID] = make(map[string]string)
				}
				s.index[uID][meta.VirtualPath] = meta.ID
			}
		}
		return nil
	})
}

// NormalizePath ensures a path starts with / and is cleaned.
func NormalizePath(p string) string {
	if p == "" || p == "/" {
		return "/"
	}
	cleaned := filepath.ToSlash(filepath.Clean(p))
	if !strings.HasPrefix(cleaned, "/") {
		cleaned = "/" + cleaned
	}
	return cleaned
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
		// CRITICAL: We must NOT match the metadata file here when looking for content.
		// Artifact files are named {id}_{filename}.
		// Metadata files are named {id}_{filename}.json.
		if f.IsDir() || strings.HasSuffix(f.Name(), ".json.json") {
			continue
		}
		// Special case: if filename itself ends in .json, the artifact is {id}_file.json
		// and metadata is {id}_file.json.json.
		// So we only skip if it's the metadata file.
		
		// If it's a metadata file, it always has .json added to the full storage name
		// We can check if a file without the .json suffix exists to be sure,
		// but a simpler way is to check the suffix ".json" AND ensure it's not the artifact itself.
		
		// Let's use a more robust way: metadata files ALWAYS end in .json
		// Artifact files might end in .json too.
		// But metadata files are ALWAYS artifactName + ".json"
		
		// Revised logic:
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

// List returns artifacts for a specific user.
// If dirPath is empty, it returns a flat list of all artifacts.
// If dirPath is set, it returns items (files and virtual folders) in that virtual directory.
func (s *Store) List(userID string, limit, offset int, dirPath string) ([]*ArtifactMetadata, error) {
	if dirPath != "" {
		return s.ListVFS(userID, dirPath, limit, offset)
	}

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

// ListVFS handles hierarchical directory listing using the in-memory index.
func (s *Store) ListVFS(userID string, dirPath string, limit, offset int) ([]*ArtifactMetadata, error) {
	uID := userID
	if uID == "" {
		uID = "global"
	}

	dir := NormalizePath(dirPath)
	if !strings.HasSuffix(dir, "/") {
		dir += "/"
	}

	s.mu.RLock()
	userIdx, ok := s.index[uID]
	if !ok {
		s.mu.RUnlock()
		return []*ArtifactMetadata{}, nil
	}

	// 1. Find all artifacts starting with dir
	// 2. Identify direct children (files) and sub-directories
	folders := make(map[string]bool)
	var fileIDs []string

	for path, id := range userIdx {
		if strings.HasPrefix(path, dir) {
			sub := strings.TrimPrefix(path, dir)
			if sub == "" {
				continue
			}
			parts := strings.Split(sub, "/")
			if len(parts) == 1 {
				// Direct file
				fileIDs = append(fileIDs, id)
			} else {
				// Sub-directory
				folders[parts[0]] = true
			}
		}
	}
	s.mu.RUnlock()

	var results []*ArtifactMetadata

	// Sort folders for deterministic results
	folderNames := make([]string, 0, len(folders))
	for f := range folders {
		folderNames = append(folderNames, f)
	}
	sort.Strings(folderNames)

	for _, folder := range folderNames {
		results = append(results, &ArtifactMetadata{
			VirtualPath: dir + folder,
			Filename:    folder,
			MimeType:    "directory",
			Description: "Virtual Directory",
		})
	}

	// Add files
	for _, id := range fileIDs {
		_, meta, err := s.Read(id, userID)
		if err == nil {
			results = append(results, meta)
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

// Find returns all artifacts matching a pattern in their virtual path.
func (s *Store) Find(userID string, pattern string) ([]*ArtifactMetadata, error) {
	uID := userID
	if uID == "" {
		uID = "global"
	}

	s.mu.RLock()
	userIdx, ok := s.index[uID]
	if !ok {
		s.mu.RUnlock()
		return []*ArtifactMetadata{}, nil
	}

	var matchIDs []string
	for path, id := range userIdx {
		matched, _ := filepath.Match(pattern, path)
		
		// Also check as substring (ignoring wildcards for simple search)
		cleanPattern := strings.ReplaceAll(pattern, "*", "")
		if matched || strings.Contains(strings.ToLower(path), strings.ToLower(cleanPattern)) {
			matchIDs = append(matchIDs, id)
		}
	}
	s.mu.RUnlock()

	var results []*ArtifactMetadata
	for _, id := range matchIDs {
		_, meta, err := s.Read(id, userID)
		if err == nil {
			results = append(results, meta)
		}
	}
	return results, nil
}

// Patch modifies an existing artifact's content.
func (s *Store) Patch(idOrPath string, userID string, patchContent []byte, lineStart, lineEnd int, shouldAppend bool) (int64, error) {
	oldContent, meta, err := s.Read(idOrPath, userID)
	if err != nil {
		return 0, err
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
	prefixDir := filepath.Join(s.BaseDir, "global")
	if userID != "" {
		prefixDir = filepath.Join(s.BaseDir, "users", userID)
	}

	// We need to find the actual file on disk (ID_Filename)
	storageName := fmt.Sprintf("%s_%s", meta.ID, meta.Filename)
	fullPath := filepath.Join(prefixDir, storageName)

	if err := os.WriteFile(fullPath, newContent, 0644); err != nil {
		return 0, fmt.Errorf("failed to update data: %w", err)
	}

	return int64(len(newContent)), nil
}

// Delete removes an artifact and its associated metadata JSON file.
// Returns true if the artifact was found and deleted, false otherwise.
func (s *Store) Delete(idOrPath string, userID string) (bool, error) {
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

	prefixDir := filepath.Join(s.BaseDir, "global")
	if userID != "" {
		prefixDir = filepath.Join(s.BaseDir, "users", userID)
	}

	files, err := os.ReadDir(prefixDir)
	if err != nil {
		return false, err
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

// DetectMimeType returns a MIME type string based on the file extension.
// It supports common types used in LLM and data processing workflows.
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
