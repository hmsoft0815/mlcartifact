package storage

import (
	"fmt"
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestVFS_Exhaustive covers the full lifecycle of VFS operations in various scenarios.
func TestVFS_Exhaustive(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "vfs-exhaustive-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	store := NewStore(tempDir)
	userID := "power-user"

	t.Run("DeepHierarchy", func(t *testing.T) {
		// Create a file deep in the hierarchy
		path := "/a/b/c/d/e/f/file.txt"
		content := []byte("deep data")
		_, err := store.Write("file.txt", content, "", 1, "test", userID, "", nil, path)
		require.NoError(t, err)

		// Verify we can read it back
		got, _, err := store.Read(path, userID)
		require.NoError(t, err)
		assert.Equal(t, content, got)

		// Verify ListVFS at each level
		levels := []string{"/", "/a", "/a/b", "/a/b/c", "/a/b/c/d", "/a/b/c/d/e", "/a/b/c/d/e/f"}
		for _, lvl := range levels {
			items, err := store.List(userID, 10, 0, lvl)
			require.NoError(t, err)
			assert.NotEmpty(t, items, "Level %s should not be empty", lvl)
			if lvl == "/a/b/c/d/e/f" {
				assert.Equal(t, "file.txt", items[0].Filename)
			} else {
				assert.Equal(t, "directory", items[0].MimeType)
			}
		}
	})

	t.Run("NameCollisions", func(t *testing.T) {
		// Same filename in different directories
		p1 := "/folder1/config.json"
		p2 := "/folder2/config.json"
		c1 := []byte(`{"id": 1}`)
		c2 := []byte(`{"id": 2}`)

		_, err := store.Write("config.json", c1, "", 1, "test", userID, "", nil, p1)
		require.NoError(t, err)
		_, err = store.Write("config.json", c2, "", 1, "test", userID, "", nil, p2)
		require.NoError(t, err)

		// Ensure they are distinct
		got1, _, _ := store.Read(p1, userID)
		got2, _, _ := store.Read(p2, userID)
		assert.Equal(t, c1, got1)
		assert.Equal(t, c2, got2)
		assert.NotEqual(t, got1, got2)
	})

	t.Run("SpecialCharacters", func(t *testing.T) {
		path := "/My Projects/🚀 Space/data @ v1.txt"
		content := []byte("special")
		_, err := store.Write("data.txt", content, "", 1, "test", userID, "", nil, path)
		require.NoError(t, err)

		got, _, err := store.Read(path, userID)
		require.NoError(t, err)
		assert.Equal(t, content, got)
	})

	t.Run("PatchEdgeCases", func(t *testing.T) {
		path := "/patch/test.txt"
		initial := []byte("line1\nline2\nline3")
		_, _ = store.Write("test.txt", initial, "", 1, "test", userID, "", nil, path)

		// 1. Replace first line
		_, err := store.Patch(path, userID, []byte("NEW1"), 0, 1, false)
		require.NoError(t, err)
		got, _, _ := store.Read(path, userID)
		assert.Equal(t, "NEW1\nline2\nline3", string(got))

		// 2. Replace middle line
		_, err = store.Patch(path, userID, []byte("NEW2"), 1, 2, false)
		require.NoError(t, err)
		got, _, _ = store.Read(path, userID)
		assert.Equal(t, "NEW1\nNEW2\nline3", string(got))

		// 3. Replace last line
		_, err = store.Patch(path, userID, []byte("NEW3"), 2, 3, false)
		require.NoError(t, err)
		got, _, _ = store.Read(path, userID)
		assert.Equal(t, "NEW1\nNEW2\nNEW3", string(got))

		// 4. Out of bounds (start > len) -> should append
		_, err = store.Patch(path, userID, []byte("EXTRA"), 10, 11, false)
		require.NoError(t, err)
		got, _, _ = store.Read(path, userID)
		assert.Contains(t, string(got), "NEW3\nEXTRA")

		// 5. Empty file patch
		emptyPath := "/patch/empty.txt"
		_, _ = store.Write("empty.txt", []byte(""), "", 1, "test", userID, "", nil, emptyPath)
		_, err = store.Patch(emptyPath, userID, []byte("content"), 0, 0, false)
		require.NoError(t, err)
		got, _, _ = store.Read(emptyPath, userID)
		assert.Equal(t, "content", string(got))
	})

	t.Run("Concurrency", func(t *testing.T) {
		path := "/concurrent/shared.txt"
		_, _ = store.Write("shared.txt", []byte("start"), "", 1, "test", userID, "", nil, path)

		var wg sync.WaitGroup
		numWorkers := 20
		wg.Add(numWorkers)

		for i := 0; i < numWorkers; i++ {
			go func(idx int) {
				defer wg.Done()
				// Mix of reads and patches
				if idx%2 == 0 {
					_, _, _ = store.Read(path, userID)
				} else {
					_, _ = store.Patch(path, userID, []byte(fmt.Sprintf("\nline %d", idx)), 0, 0, true)
				}
			}(i)
		}
		wg.Wait()

		// Final read to ensure lock didn't fail and data is consistent (not corrupted)
		got, _, err := store.Read(path, userID)
		require.NoError(t, err)
		assert.NotEmpty(t, got)
	})

	t.Run("FindComplex", func(t *testing.T) {
		// Setup files
		store.Write("f1.txt", []byte("1"), "", 1, "t", userID, "", nil, "/logs/2026-03-17.log")
		store.Write("f2.txt", []byte("2"), "", 1, "t", userID, "", nil, "/logs/2026-03-18.log")
		store.Write("f3.txt", []byte("3"), "", 1, "t", userID, "", nil, "/reports/final.pdf")

		// 1. Glob match
		items, _ := store.Find(userID, "/logs/*.log")
		assert.Len(t, items, 2)

		// 2. Substring match
		items, _ = store.Find(userID, "final")
		assert.Len(t, items, 1)
		assert.Equal(t, "f3.txt", items[0].Filename)

		// 3. No match
		items, _ = store.Find(userID, "non-existent")
		assert.Empty(t, items)
	})

	t.Run("UserIsolation", func(t *testing.T) {
		path := "/shared/secret.txt"
		_, _ = store.Write("s.txt", []byte("private"), "", 1, "t", "user-a", "", nil, path)

		// Try to read from user-b
		_, _, err := store.Read(path, "user-b")
		assert.Error(t, err, "User B should not see User A's virtual path")

		// List root for user-b
		items, _ := store.List("user-b", 10, 0, "/")
		assert.Empty(t, items)
	})
}

// TestVFS_Persistence specifically checks index reconstruction.
func TestVFS_Persistence(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "vfs-persistence-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	uID := "persistent-user"
	path := "/data/save.txt"
	
	// 1. Write data
	s1 := NewStore(tempDir)
	_, err = s1.Write("save.txt", []byte("payload"), "", 1, "test", uID, "", nil, path)
	require.NoError(t, err)

	// 2. Simulate server restart by creating new store instance
	s2 := NewStore(tempDir)

	// 3. Verify path is still valid
	got, meta, err := s2.Read(path, uID)
	require.NoError(t, err)
	assert.Equal(t, []byte("payload"), got)
	assert.Equal(t, path, meta.VirtualPath)

	// 4. Verify Delete updates index across restarts
	_, _ = s2.Delete(path, uID)
	s3 := NewStore(tempDir)
	_, _, err = s3.Read(path, uID)
	assert.Error(t, err, "Path should remain deleted after restart")
}
