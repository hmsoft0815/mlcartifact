package grpc

import (
	"context"
	"os"
	"testing"

	"github.com/mlcmcp/artifact-server/internal/storage"
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
	assert.Contains(t, writeRes.Uri, writeReq.Filename)

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
	s.Write(ctx, &pb.WriteRequest{Filename: "f1", Content: []byte("1"), UserId: userId})
	s.Write(ctx, &pb.WriteRequest{Filename: "f2", Content: []byte("2"), UserId: userId})

	// Test List
	listRes, err := s.List(ctx, &pb.ListRequest{UserId: userId})
	require.NoError(t, err)
	assert.Len(t, listRes.Items, 2)

	// Test Delete
	delRes, err := s.Delete(ctx, &pb.DeleteRequest{Id: listRes.Items[0].Id, UserId: userId})
	require.NoError(t, err)
	assert.True(t, delRes.Deleted)

	// List again
	listRes2, _ := s.List(ctx, &pb.ListRequest{UserId: userId})
	assert.Len(t, listRes2.Items, 1)
}
