package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStore_WriteRead(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "artifact-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	store := NewStore(tempDir)
	content := []byte("hello world")
	filename := "test.txt"
	mimeType := "text/plain"
	source := "unit-test"
	userID := "user123"
	metadata := map[string]interface{}{"key": "value"}

	// Test Write
	meta, err := store.Write(filename, content, mimeType, 1, source, userID, "test description", metadata, "")
	require.NoError(t, err)
	assert.NotEmpty(t, meta.ID)
	assert.Equal(t, filename, meta.Filename)
	assert.Equal(t, mimeType, meta.MimeType)
	assert.Equal(t, userID, meta.UserID)

	// Test Read by ID
	readData, readMeta, err := store.Read(meta.ID, userID)
	require.NoError(t, err)
	assert.Equal(t, content, readData)
	assert.Equal(t, meta.ID, readMeta.ID)
	assert.Equal(t, "test description", readMeta.Description)
	assert.Equal(t, metadata["key"], readMeta.Metadata["key"])

	// Test Read by ID (wrong user should fail)
	_, _, err = store.Read(meta.ID, "wrong-user")
	assert.Error(t, err)

	// Test Read by Filename (prefix)
	readData2, _, err := store.Read(meta.Filename, userID)
	require.NoError(t, err)
	assert.Equal(t, content, readData2)
}

func TestStore_List(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "artifact-list-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	store := NewStore(tempDir)
	userID := "user456"

	_, _ = store.Write("file1.txt", []byte("1"), "", 1, "src", userID, "", nil, "")
	_, _ = store.Write("file2.txt", []byte("2"), "", 1, "src", userID, "", nil, "")
	_, _ = store.Write("global.txt", []byte("3"), "", 1, "src", "", "", nil, "")

	// List user artifacts
	items, err := store.List(userID, 100, 0, "")
	require.NoError(t, err)
	assert.Len(t, items, 2)

	// List global artifacts
	globalItems, err := store.List("", 100, 0, "")
	require.NoError(t, err)
	assert.Len(t, globalItems, 1)

	// Test Pagination
	// Create more global artifacts
	for i := 0; i < 5; i++ {
		_, _ = store.Write(fmt.Sprintf("paginated-%d.txt", i), []byte("test"), "", 1, "src", "", "", nil, "")
	}

	pItems, err := store.List("", 2, 0, "") // limit 2, offset 0
	require.NoError(t, err)
	assert.Len(t, pItems, 2)

	pItems2, err := store.List("", 2, 2, "") // limit 2, offset 2
	require.NoError(t, err)
	assert.Len(t, pItems2, 2)
	assert.NotEqual(t, pItems[0].ID, pItems2[0].ID)

	pItems3, err := store.List("", 10, 5, "") // limit 10, offset 5
	require.NoError(t, err)
	assert.Len(t, pItems3, 1) // 6 global files total, offset 5 -> only 1 left
}

func TestStore_VFS(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "artifact-vfs-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	store := NewStore(tempDir)
	userID := "vfs-user"

	// 1. Write with virtual path
	path := "/projects/alpha/readme.md"
	content := []byte("# Project Alpha")
	meta, err := store.Write("readme.md", content, "text/markdown", 1, "test", userID, "", nil, path)
	require.NoError(t, err)
	assert.Equal(t, path, meta.VirtualPath)

	// 2. Read by virtual path
	readData, readMeta, err := store.Read(path, userID)
	require.NoError(t, err)
	assert.Equal(t, content, readData)
	assert.Equal(t, path, readMeta.VirtualPath)

	// 3. List VFS (root)
	// Add another file in different folder
	_, _ = store.Write("other.txt", []byte("other"), "", 1, "test", userID, "", nil, "/docs/manual.txt")
	// Add another file in same folder
	_, _ = store.Write("todo.md", []byte("todo"), "", 1, "test", userID, "", nil, "/projects/alpha/todo.md")

	// List /
	rootItems, err := store.List(userID, 100, 0, "/")
	require.NoError(t, err)
	// Should contain "projects" and "docs" folders
	assert.Equal(t, 2, len(rootItems))
	
	foundProjects := false
	foundDocs := false
	for _, item := range rootItems {
		if item.Filename == "projects" && item.MimeType == "directory" {
			foundProjects = true
		}
		if item.Filename == "docs" && item.MimeType == "directory" {
			foundDocs = true
		}
	}
	assert.True(t, foundProjects, "projects folder should be found in /")
	assert.True(t, foundDocs, "docs folder should be found in /")

	// List /projects/alpha/
	alphaItems, err := store.List(userID, 100, 0, "/projects/alpha")
	require.NoError(t, err)
	// Should contain "readme.md" and "todo.md" files
	assert.Len(t, alphaItems, 2)
	assert.NotEqual(t, "directory", alphaItems[0].MimeType)

	// 4. Find
	findItems, err := store.Find(userID, "*readme*")
	require.NoError(t, err)
	assert.Len(t, findItems, 1)
	assert.Equal(t, "readme.md", findItems[0].Filename)

	// 5. Patch (Append)
	newSize, err := store.Patch(path, userID, []byte("\n- Task 1"), 0, 0, true)
	require.NoError(t, err)
	assert.Greater(t, newSize, int64(len(content)))

	patchedData, _, _ := store.Read(path, userID)
	assert.Contains(t, string(patchedData), "- Task 1")

	// 6. Patch (Line replacement)
	// Original: "# Project Alpha\n- Task 1"
	// Replace line 0 (index 0)
	_, err = store.Patch(path, userID, []byte("# NEW TITLE"), 0, 1, false)
	require.NoError(t, err)
	patchedData2, _, _ := store.Read(path, userID)
	assert.True(t, strings.HasPrefix(string(patchedData2), "# NEW TITLE"))
	assert.Contains(t, string(patchedData2), "- Task 1")

	// 7. Delete by path
	deleted, err := store.Delete(path, userID)
	require.NoError(t, err)
	assert.True(t, deleted)

	// Verify gone from index
	_, _, err = store.Read(path, userID)
	assert.Error(t, err)
}

func TestStore_IndexRebuild(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "artifact-index-rebuild-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// 1. Create a store and write a file
	store1 := NewStore(tempDir)
	path := "/persistent/file.txt"
	_, err = store1.Write("file.txt", []byte("data"), "", 1, "test", "user1", "", nil, path)
	require.NoError(t, err)

	// 2. Create a new store instance pointing to same dir
	store2 := NewStore(tempDir)
	
	// 3. Verify it found the file in index
	_, _, err = store2.Read(path, "user1")
	require.NoError(t, err, "New store instance should rebuild index from disk")
}

func TestDetectMimeType(t *testing.T) {
	assert.Equal(t, "text/markdown", DetectMimeType("readme.md"))
	assert.Equal(t, "image/svg+xml", DetectMimeType("logo.svg"))
	assert.Equal(t, "application/octet-stream", DetectMimeType("random.dat"))
}

func TestStore_UserIDValidation_Security(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "artifact-sec-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	store := NewStore(tempDir)

	// Valid user IDs
	validIDs := []string{"", "u1", "user-123", "User_456", "550e8400-e29b-41d4-a716-446655440000"}
	for _, id := range validIDs {
		assert.NoError(t, ValidateUserID(id), "expected %q to be valid", id)
	}

	// Invalid user IDs (traversal, paths, illegal chars)
	invalidIDs := []string{
		"..",
		".",
		"../outside",
		"../../outside",
		"/root",
		`\windows`,
		"user/sub",
		`user\sub`,
		"user id",
		"user\x00id",
		"user.name",
		strings.Repeat("a", 129),
	}
	for _, id := range invalidIDs {
		assert.ErrorIs(t, ValidateUserID(id), ErrInvalidUserID, "expected %q to be invalid", id)
	}

	// Verify all operations reject traversal user IDs
	badUser := "../../traversal"

	// 1. Write
	_, err = store.Write("evil.txt", []byte("evil"), "", 1, "test", badUser, "", nil, "")
	assert.ErrorIs(t, err, ErrInvalidUserID)
	// Ensure no file was written outside baseDir
	_, err = os.Stat(filepath.Join(tempDir, "..", "traversal"))
	assert.True(t, os.IsNotExist(err), "directory should not be created outside baseDir")

	// 2. Read
	_, _, err = store.Read("any-id", badUser)
	assert.ErrorIs(t, err, ErrInvalidUserID)

	// 3. List
	_, err = store.List(badUser, 10, 0, "")
	assert.ErrorIs(t, err, ErrInvalidUserID)

	// 4. ListVFS
	_, err = store.List(badUser, 10, 0, "/")
	assert.ErrorIs(t, err, ErrInvalidUserID)

	// 5. Patch
	_, err = store.Patch("any-id", badUser, []byte("x"), 0, 0, true)
	assert.ErrorIs(t, err, ErrInvalidUserID)

	// 6. Delete
	_, err = store.Delete("any-id", badUser)
	assert.ErrorIs(t, err, ErrInvalidUserID)

	// 7. Find
	_, err = store.Find(badUser, "*")
	assert.ErrorIs(t, err, ErrInvalidUserID)
}

func TestStore_NonExistentUser_NoErrorLeak(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "artifact-leak-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	store := NewStore(tempDir)
	nonExistentUser := "user-does-not-exist"

	// Read should return ErrArtifactNotFound, NOT os.PathError
	_, _, err = store.Read("missing-artifact", nonExistentUser)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrArtifactNotFound)
	assert.NotContains(t, err.Error(), tempDir, "error message must not contain server base directory")
	assert.NotContains(t, err.Error(), "no such file or directory")

	// Delete should return false, nil
	deleted, err := store.Delete("missing-artifact", nonExistentUser)
	require.NoError(t, err)
	assert.False(t, deleted)

	// List should return empty slice, nil
	items, err := store.List(nonExistentUser, 10, 0, "")
	require.NoError(t, err)
	assert.Empty(t, items)
}
