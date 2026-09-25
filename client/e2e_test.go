// Copyright (c) 2026 Michael Lechner. All rights reserved.
// Use of this source code is governed by the MIT license that can be
// found in the LICENSE file.

package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hmsoft0815/mlcartifact/internal/grpc"
	"github.com/hmsoft0815/mlcartifact/internal/storage"
	"github.com/hmsoft0815/mlcartifact/proto/protoconnect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestClientEndToEnd drives every RPC against a real in-process server.
func TestClientEndToEnd(t *testing.T) {
	t.Setenv("ARTIFACT_USER_ID", "")
	mux := http.NewServeMux()
	path, handler := protoconnect.NewArtifactServiceHandler(grpc.NewConnectServer(storage.NewStore(t.TempDir())))
	mux.Handle(path, handler)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c, err := NewClientWithAddr(srv.URL, WithHTTPClient(srv.Client()))
	require.NoError(t, err)
	ctx := context.Background()

	_, err = c.Write(ctx, "other.txt", []byte("x"), WithSource("someone-else"))
	require.NoError(t, err)

	w, err := c.Write(ctx, "notes.md", []byte("a\nb\nc"), WithVirtualPath("/proj/docs/notes.md"), WithSource("e2e"))
	require.NoError(t, err)
	assert.Equal(t, "/proj/docs/notes.md", w.VirtualPath)

	p, err := c.Patch(ctx, "/proj/docs/notes.md", []byte("B"), WithLines(1, 2))
	require.NoError(t, err)
	assert.True(t, p.Success)
	_, err = c.Patch(ctx, w.Id, []byte("\nd"), WithAppend())
	require.NoError(t, err)

	r, err := c.Read(ctx, "/proj/docs/notes.md")
	require.NoError(t, err)
	assert.Equal(t, "a\nB\nc\nd", string(r.Content))

	dir, err := c.List(ctx, "", WithDirPath("/proj"))
	require.NoError(t, err)
	require.NotEmpty(t, dir.Items)
	assert.True(t, dir.Items[0].IsDirectory, "expected /proj/docs as a directory entry")

	bySource, err := c.List(ctx, "", WithSourceFilter("e2e"))
	require.NoError(t, err)
	require.Len(t, bySource.Items, 1)
	assert.Equal(t, "notes.md", bySource.Items[0].Filename)

	found, err := c.Find(ctx, "*.md")
	require.NoError(t, err)
	assert.Len(t, found.Items, 1)

	d, err := c.Delete(ctx, "/proj/docs/notes.md")
	require.NoError(t, err)
	assert.True(t, d.Deleted)
}
