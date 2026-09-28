package grpc

import (
	"context"
	"fmt"
	"os"
	"testing"

	"connectrpc.com/connect"
	"github.com/hmsoft0815/mlcartifact/internal/storage"
	pb "github.com/hmsoft0815/mlcartifact/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServer_WriteRead(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "artifact-grpc-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	store := storage.NewStore(tempDir)
	s := NewServer(store)
	ctx := context.Background()

	// Test Write
	writeReq := &pb.WriteRequest{
		Filename:     "test.txt",
		Content:      []byte("grpc data"),
		MimeType:     "text/plain",
		ExpiresHours: 1,
		Source:       "grpc-test",
		UserId:       "test-user",
		Metadata:     map[string]string{"foo": "bar"},
	}

	writeRes, err := s.Write(ctx, writeReq)
	require.NoError(t, err)
	assert.NotEmpty(t, writeRes.Id)
	assert.Equal(t, fmt.Sprintf("mlcartifact://%s", writeRes.Id), writeRes.Uri)

	// Test Read
	readReq := &pb.ReadRequest{
		Id:     writeRes.Id,
		UserId: "test-user",
	}

	readRes, err := s.Read(ctx, readReq)
	require.NoError(t, err)
	assert.Equal(t, writeReq.Content, readRes.Content)
	assert.Equal(t, writeReq.Filename, readRes.Filename)
}

func TestServer_ListDelete(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "artifact-grpc-list-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	store := storage.NewStore(tempDir)
	s := NewServer(store)
	ctx := context.Background()

	userId := "list-user"
	_, err = s.Write(ctx, &pb.WriteRequest{Filename: "f1", Content: []byte("12345"), UserId: userId})
	require.NoError(t, err)
	_, err = s.Write(ctx, &pb.WriteRequest{Filename: "f2", Content: []byte("12"), UserId: userId})
	require.NoError(t, err)

	// Test List
	listRes, err := s.List(ctx, &pb.ListRequest{UserId: userId})
	require.NoError(t, err)
	assert.Len(t, listRes.Items, 2)
	sizes := map[string]int64{}
	for _, item := range listRes.Items {
		sizes[item.Filename] = item.SizeBytes
	}
	assert.Equal(t, int64(5), sizes["f1"])
	assert.Equal(t, int64(2), sizes["f2"])

	// Test Delete
	delRes, err := s.Delete(ctx, &pb.DeleteRequest{Id: listRes.Items[0].Id, UserId: userId})
	require.NoError(t, err)
	assert.True(t, delRes.Deleted)

	// List again
	listRes2, _ := s.List(ctx, &pb.ListRequest{UserId: userId})
	assert.Len(t, listRes2.Items, 1)
}

func TestServer_Security_InvalidUserID(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "artifact-grpc-sec-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	store := storage.NewStore(tempDir)
	s := NewServer(store)
	ctx := context.Background()

	badUser := "../../evil-user"

	// 1. Write
	_, err = s.Write(ctx, &pb.WriteRequest{Filename: "evil.txt", Content: []byte("hack"), UserId: badUser})
	require.Error(t, err)
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	assert.NotContains(t, err.Error(), tempDir)

	// 2. Read
	_, err = s.Read(ctx, &pb.ReadRequest{Id: "any-id", UserId: badUser})
	require.Error(t, err)
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	assert.NotContains(t, err.Error(), tempDir)

	// 3. Delete
	_, err = s.Delete(ctx, &pb.DeleteRequest{Id: "any-id", UserId: badUser})
	require.Error(t, err)
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	assert.NotContains(t, err.Error(), tempDir)

	// 4. List
	_, err = s.List(ctx, &pb.ListRequest{UserId: badUser})
	require.Error(t, err)
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	assert.NotContains(t, err.Error(), tempDir)

	// 5. Patch
	_, err = s.Patch(ctx, &pb.PatchRequest{Id: "any-id", UserId: badUser, Content: []byte("patch")})
	require.Error(t, err)
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	assert.NotContains(t, err.Error(), tempDir)

	// 6. Find
	_, err = s.Find(ctx, &pb.FindRequest{UserId: badUser, Pattern: "*"})
	require.Error(t, err)
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	assert.NotContains(t, err.Error(), tempDir)
}

func TestServer_Security_NonExistentUser_NoErrorLeak(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "artifact-grpc-leak-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	store := storage.NewStore(tempDir)
	s := NewServer(store)
	ctx := context.Background()

	nonExistentUser := "user-u2"

	// 1. Read: must return NotFound, NOT Internal, and MUST NOT leak server paths
	_, err = s.Read(ctx, &pb.ReadRequest{Id: "missing-id", UserId: nonExistentUser})
	require.Error(t, err)
	assert.Equal(t, connect.CodeNotFound, connect.CodeOf(err), "expected CodeNotFound for missing artifact in non-existent user dir")
	assert.NotContains(t, err.Error(), tempDir, "must not leak server storage path")
	assert.NotContains(t, err.Error(), "no such file or directory")

	// 2. Delete: must return NotFound
	_, err = s.Delete(ctx, &pb.DeleteRequest{Id: "missing-id", UserId: nonExistentUser})
	require.Error(t, err)
	assert.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
	assert.NotContains(t, err.Error(), tempDir)

	// 3. List: must return empty list without error
	listRes, err := s.List(ctx, &pb.ListRequest{UserId: nonExistentUser})
	require.NoError(t, err)
	assert.Empty(t, listRes.Items)
}

func TestServer_Patch_BinaryFile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "artifact-grpc-binary-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	store := storage.NewStore(tempDir)
	s := NewServer(store)
	ctx := context.Background()

	// Write a binary file (PNG)
	writeRes, err := s.Write(ctx, &pb.WriteRequest{
		Filename: "image.png",
		Content:  []byte("\x89PNG\r\n\x1a\n\x00data"),
		MimeType: "image/png",
		UserId:   "user1",
	})
	require.NoError(t, err)

	// Attempt to patch the binary file
	patchRes, err := s.Patch(ctx, &pb.PatchRequest{
		Id:      writeRes.Id,
		UserId:  "user1",
		Content: []byte("new-content"),
		Append:  true,
	})
	require.Error(t, err)
	assert.Nil(t, patchRes)
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	assert.Contains(t, err.Error(), "cannot patch binary file")
}
