package mcp

import (
	"context"
	"os"
	"testing"

	"github.com/hmsoft0815/mlcartifact/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMCP_Security_InvalidUserID(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "artifact-mcp-sec-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	s := storage.NewStore(tempDir)
	SetStore(s)
	ctx := context.Background()

	badUser := "../../evil-user"

	// 1. WriteArtifact
	_, _, err = WriteArtifact(ctx, nil, WriteArtifactArgs{
		Filename: "evil.txt",
		Content:  "content",
		UserID:   badUser,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid user_id")
	assert.NotContains(t, err.Error(), tempDir)

	// 2. ReadArtifact
	_, _, err = ReadArtifact(ctx, nil, ReadArtifactArgs{
		ID:     "any-id",
		UserID: badUser,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid user_id")
	assert.NotContains(t, err.Error(), tempDir)

	// 3. DeleteArtifact
	_, _, err = DeleteArtifact(ctx, nil, DeleteArtifactArgs{
		ID:     "any-id",
		UserID: badUser,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid user_id")
	assert.NotContains(t, err.Error(), tempDir)

	// 4. ListArtifacts
	_, _, err = ListArtifacts(ctx, nil, ListArtifactsArgs{
		UserID: badUser,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid user_id")
	assert.NotContains(t, err.Error(), tempDir)

	// 5. VFSPatch
	_, _, err = VFSPatch(ctx, nil, VFSPatchArgs{
		ID:      "any-id",
		Content: "patch",
		UserID:  badUser,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid user_id")
	assert.NotContains(t, err.Error(), tempDir)

	// 6. VFSList
	_, _, err = VFSList(ctx, nil, VFSListArgs{
		Path:   "/",
		UserID: badUser,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid user_id")
	assert.NotContains(t, err.Error(), tempDir)

	// 7. VFSFind
	_, _, err = VFSFind(ctx, nil, VFSFindArgs{
		Pattern: "*",
		UserID:  badUser,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid user_id")
	assert.NotContains(t, err.Error(), tempDir)
}

func TestMCP_Security_NonExistentUser_NoErrorLeak(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "artifact-mcp-leak-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	s := storage.NewStore(tempDir)
	SetStore(s)
	ctx := context.Background()

	nonExistentUser := "user-u2"

	// 1. ReadArtifact
	_, _, err = ReadArtifact(ctx, nil, ReadArtifactArgs{
		ID:     "missing-id",
		UserID: nonExistentUser,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
	assert.NotContains(t, err.Error(), tempDir)
	assert.NotContains(t, err.Error(), "no such file or directory")

	// 2. DeleteArtifact
	_, _, err = DeleteArtifact(ctx, nil, DeleteArtifactArgs{
		ID:     "missing-id",
		UserID: nonExistentUser,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
	assert.NotContains(t, err.Error(), tempDir)
	assert.NotContains(t, err.Error(), "no such file or directory")

	// 3. ListArtifacts: returns empty list, no error
	_, list, err := ListArtifacts(ctx, nil, ListArtifactsArgs{
		UserID: nonExistentUser,
	})
	require.NoError(t, err)
	assert.Empty(t, list.Artifacts)
}

func TestMCP_VFSPatch_BinaryFile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "artifact-mcp-binary-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	s := storage.NewStore(tempDir)
	SetStore(s)
	ctx := context.Background()

	meta, err := s.Write("photo.jpg", []byte("\xff\xd8\xff\xe0data"), "image/jpeg", 1, "test", "user1", "", nil, "")
	require.NoError(t, err)

	_, _, err = VFSPatch(ctx, nil, VFSPatchArgs{
		ID:      meta.ID,
		UserID:  "user1",
		Content: "text patch",
		Append:  true,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot patch binary file")
}

