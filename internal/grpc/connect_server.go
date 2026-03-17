// Copyright (c) 2026 Michael Lechner. All rights reserved.
package grpc

import (
	"context"

	connect "connectrpc.com/connect"
	"github.com/hmsoft0815/mlcartifact/internal/storage"
	pb "github.com/hmsoft0815/mlcartifact/proto"
	"github.com/hmsoft0815/mlcartifact/proto/protoconnect"
)

// ConnectServer implements the Connect interface for ArtifactService.
type ConnectServer struct {
	server *Server
}

// NewConnectServer creates a new ConnectServer.
func NewConnectServer(store *storage.Store) protoconnect.ArtifactServiceHandler {
	return &ConnectServer{
		server: NewServer(store),
	}
}

func (c *ConnectServer) Write(ctx context.Context, req *connect.Request[pb.WriteRequest]) (*connect.Response[pb.WriteResponse], error) {
	res, err := c.server.Write(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(res), nil
}

func (c *ConnectServer) Read(ctx context.Context, req *connect.Request[pb.ReadRequest]) (*connect.Response[pb.ReadResponse], error) {
	res, err := c.server.Read(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(res), nil
}

func (c *ConnectServer) Delete(ctx context.Context, req *connect.Request[pb.DeleteRequest]) (*connect.Response[pb.DeleteResponse], error) {
	res, err := c.server.Delete(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(res), nil
}

func (c *ConnectServer) List(ctx context.Context, req *connect.Request[pb.ListRequest]) (*connect.Response[pb.ListResponse], error) {
	res, err := c.server.List(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(res), nil
}

func (c *ConnectServer) Patch(ctx context.Context, req *connect.Request[pb.PatchRequest]) (*connect.Response[pb.PatchResponse], error) {
	res, err := c.server.Patch(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(res), nil
}

func (c *ConnectServer) Find(ctx context.Context, req *connect.Request[pb.FindRequest]) (*connect.Response[pb.ListResponse], error) {
	res, err := c.server.Find(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(res), nil
}
