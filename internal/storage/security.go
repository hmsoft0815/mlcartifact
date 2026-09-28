// Copyright (c) 2026 Michael Lechner. All rights reserved.

package storage

import (
	"errors"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

var (
	// ErrInvalidUserID is returned when a userID contains illegal characters, path separators, or exceeds max length.
	ErrInvalidUserID = errors.New("invalid user_id: must contain only alphanumeric characters, dashes, and underscores (max 128 characters)")

	// ErrArtifactNotFound is returned when the requested artifact does not exist.
	ErrArtifactNotFound = errors.New("artifact not found")

	// ErrPathEscape is returned when a storage path attempts to escape the store's base directory.
	ErrPathEscape = errors.New("invalid path: path escapes base directory")

	// ErrBinaryFile is returned when a patch operation is attempted on a binary artifact.
	ErrBinaryFile = errors.New("cannot patch binary file: patch is only supported for text files")
)

// utf8Valid reports whether s is a valid UTF-8 string.
func utf8Valid(s string) bool {
	return utf8.ValidString(s)
}

// ValidateUserID checks that a userID contains only allowed characters ([A-Za-z0-9_-])
// and does not exceed a maximum length of 128 characters.
// An empty userID is allowed and represents the global scope.
func ValidateUserID(userID string) error {
	if userID == "" {
		return nil
	}
	if len(userID) > 128 {
		return ErrInvalidUserID
	}
	for i := 0; i < len(userID); i++ {
		c := userID[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_' {
			continue
		}
		return ErrInvalidUserID
	}
	return nil
}

// userDir resolves and validates the directory path for a given userID.
// It enforces that userID is valid and that the target directory lies strictly within BaseDir.
func (s *Store) userDir(userID string) (string, error) {
	if err := ValidateUserID(userID); err != nil {
		return "", err
	}

	cleanBase, err := filepath.Abs(s.BaseDir)
	if err != nil {
		cleanBase = filepath.Clean(s.BaseDir)
	}

	var targetDir string
	if userID == "" {
		targetDir = filepath.Join(cleanBase, "global")
	} else {
		targetDir = filepath.Join(cleanBase, "users", userID)
	}
	targetDir = filepath.Clean(targetDir)

	// Second line of defense: verify targetDir is strictly within cleanBase
	rel, err := filepath.Rel(cleanBase, targetDir)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", ErrPathEscape
	}

	return targetDir, nil
}
