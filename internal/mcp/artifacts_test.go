package mcp

import (
	"context"
	"testing"

	"github.com/hmsoft0815/mlcartifact/internal/auth"
	"github.com/hmsoft0815/mlcartifact/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMCP_Security_InvalidUserID(t *testing.T) {
	tempDir := t.TempDir()
	s := storage.NewStore(tempDir)
	SetStore(s)
	ctx := context.Background()

	badUser := "../../evil-user"

	// 1. WriteArtifact
	_, _, err := WriteArtifact(ctx, nil, WriteArtifactArgs{
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
	tempDir := t.TempDir()
	s := storage.NewStore(tempDir)
	SetStore(s)
	ctx := context.Background()

	nonExistentUser := "user-u2"

	// 1. ReadArtifact
	_, _, err := ReadArtifact(ctx, nil, ReadArtifactArgs{
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
	tempDir := t.TempDir()
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

func TestMCP_UserID_EnforcedFromAuthContext(t *testing.T) {
	tempDir := t.TempDir()
	s := storage.NewStore(tempDir)
	SetStore(s)

	id := &auth.AuthIdentity{UserID: "authenticated-user"}
	authCtx := auth.WithAuthIdentity(context.Background(), id)

	// User writes artifact with args.UserID = "spoofed-user"
	// Should be saved under "authenticated-user"
	_, res, err := WriteArtifact(authCtx, nil, WriteArtifactArgs{
		Filename: "auth-test.txt",
		Content:  "secret-content",
		UserID:   "spoofed-user",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, res.ID)

	// Reading with spoofed-user should fail (not found)
	_, _, err = ReadArtifact(context.Background(), nil, ReadArtifactArgs{
		ID:     res.ID,
		UserID: "spoofed-user",
	})
	require.Error(t, err)

	// Reading with authCtx (authenticated-user) succeeds even if args.UserID is empty or wrong
	_, content, err := ReadArtifact(authCtx, nil, ReadArtifactArgs{
		ID:     res.ID,
		UserID: "spoofed-user",
	})
	require.NoError(t, err)
	assert.Nil(t, content) // ReadArtifact returns textResult in mcp.CallToolResult

	// ListArtifacts with authCtx should show the artifact
	_, list, err := ListArtifacts(authCtx, nil, ListArtifactsArgs{
		UserID: "spoofed-user",
	})
	require.NoError(t, err)
	assert.Len(t, list.Artifacts, 1)
	assert.Equal(t, "auth-test.txt", list.Artifacts[0].Filename)
}


