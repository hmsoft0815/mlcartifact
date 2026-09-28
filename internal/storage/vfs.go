// Copyright (c) 2026 Michael Lechner. All rights reserved.

package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

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
				if uID != "" && ValidateUserID(uID) != nil {
					return nil
				}
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

// ListVFS handles hierarchical directory listing using the in-memory index.
func (s *Store) ListVFS(userID string, dirPath string, limit, offset int) ([]*ArtifactMetadata, error) {
	if err := ValidateUserID(userID); err != nil {
		return nil, err
	}

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
	if err := ValidateUserID(userID); err != nil {
		return nil, err
	}

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
