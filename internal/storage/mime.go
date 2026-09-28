// Copyright (c) 2026 Michael Lechner. All rights reserved.

package storage

import (
	"bytes"
	"mime"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// DetectMimeType returns a MIME type string based on the file extension.
// It supports common types used in LLM and data processing workflows,
// including documents, data formats, and common image formats.
func DetectMimeType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	// Documents & text
	case ".md":
		return "text/markdown"
	case ".html", ".htm":
		return "text/html"
	case ".json":
		return "application/json"
	case ".xml":
		return "application/xml"
	case ".pdf":
		return "application/pdf"
	case ".csv":
		return "text/csv"
	case ".txt", ".log":
		return "text/plain"
	case ".yaml", ".yml":
		return "text/yaml"

	// Scripts & code
	case ".js", ".mjs", ".cjs":
		return "application/javascript"
	case ".ts":
		return "application/x-typescript"

	// Image formats
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".svg":
		return "image/svg+xml"
	case ".bmp":
		return "image/bmp"
	case ".ico":
		return "image/x-icon"
	case ".tiff", ".tif":
		return "image/tiff"
	case ".avif":
		return "image/avif"
	case ".heic":
		return "image/heic"
	case ".heif":
		return "image/heif"

	default:
		if mt := mime.TypeByExtension(ext); mt != "" {
			if idx := strings.Index(mt, ";"); idx != -1 {
				mt = strings.TrimSpace(mt[:idx])
			}
			return mt
		}
		return "application/octet-stream"
	}
}

// binaryExtensions lists common file extensions that represent non-text binary files.
var binaryExtensions = map[string]bool{
	".png":   true,
	".jpg":   true,
	".jpeg":  true,
	".gif":   true,
	".webp":  true,
	".bmp":   true,
	".ico":   true,
	".tiff":  true,
	".tif":   true,
	".avif":  true,
	".heic":  true,
	".heif":  true,
	".pdf":   true,
	".zip":   true,
	".gz":    true,
	".tar":   true,
	".bz2":   true,
	".xz":    true,
	".7z":    true,
	".rar":   true,
	".exe":   true,
	".dll":   true,
	".so":    true,
	".dylib": true,
	".bin":   true,
	".iso":   true,
	".dmg":   true,
	".wasm":  true,
	".class": true,
	".o":     true,
	".a":     true,
	".mp3":   true,
	".mp4":   true,
	".wav":   true,
	".ogg":   true,
	".avi":   true,
	".mov":   true,
	".mkv":   true,
	".webm":  true,
	".flac":  true,
	".aac":   true,
}

// IsBinary reports whether an artifact is a binary file (not a text file).
// It inspects the filename extension, MIME type, and content (looking for NUL bytes or invalid UTF-8).
func IsBinary(filename, mimeType string, content []byte) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	if binaryExtensions[ext] {
		return true
	}

	cleanMime := strings.ToLower(strings.TrimSpace(mimeType))
	if idx := strings.Index(cleanMime, ";"); idx != -1 {
		cleanMime = strings.TrimSpace(cleanMime[:idx])
	}

	if strings.HasPrefix(cleanMime, "image/") && cleanMime != "image/svg+xml" {
		return true
	}
	if strings.HasPrefix(cleanMime, "audio/") || strings.HasPrefix(cleanMime, "video/") {
		return true
	}
	switch cleanMime {
	case "application/pdf",
		"application/zip",
		"application/gzip",
		"application/x-tar",
		"application/x-bzip2",
		"application/x-7z-compressed",
		"application/x-rar-compressed",
		"application/vnd.rar",
		"application/wasm",
		"application/x-executable",
		"application/x-mach-binary",
		"application/x-dosexec",
		"application/x-sharedlib":
		return true
	}

	// Content inspection: if content contains NUL byte (0x00) or invalid UTF-8, it is binary.
	if bytes.IndexByte(content, 0) != -1 || !utf8.Valid(content) {
		return true
	}

	return false
}
