// Copyright (c) 2026 Michael Lechner. All rights reserved.
package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/hmsoft0815/mlcartifact/internal/grpc"
	artifactmcp "github.com/hmsoft0815/mlcartifact/internal/mcp"
	"github.com/hmsoft0815/mlcartifact/internal/storage"
	"github.com/hmsoft0815/mlcartifact/proto/protoconnect"
	"github.com/rs/cors"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var (
	version = "dev"
	name    = "artifact-server"
)

func newServer() *mcp.Server {
	s := mcp.NewServer(
		&mcp.Implementation{Name: name, Title: "mlcartifact", Version: version},
		&mcp.ServerOptions{Instructions: artifactmcp.Instructions},
	)
	artifactmcp.Register(s)
	return s
}

// dumpTools lists the registered tools through an in-memory client session,
// so the output is exactly what a real client sees.
func dumpTools(ctx context.Context, s *mcp.Server) error {
	ct, st := mcp.NewInMemoryTransports()
	if _, err := s.Connect(ctx, st, nil); err != nil {
		return err
	}
	cs, err := mcp.NewClient(&mcp.Implementation{Name: "dump", Version: version}, nil).Connect(ctx, ct, nil)
	if err != nil {
		return err
	}
	defer cs.Close()
	res, err := cs.ListTools(ctx, nil)
	if err != nil {
		return err
	}
	b, _ := json.MarshalIndent(res.Tools, "", "  ")
	fmt.Println(string(b))
	return nil
}

// decodeBase64Headers decodes Mcp-Name and Mcp-Param-* header values sent in
// the =?base64?...?= form of spec 2026-07-28. go-sdk v1.8.0 compares the raw
// value with the body and would reject such requests with -32020.
func decodeBase64Headers(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for key, values := range r.Header {
			if key != "Mcp-Name" && !strings.HasPrefix(key, "Mcp-Param-") {
				continue
			}
			for i, v := range values {
				enc, ok := strings.CutPrefix(v, "=?base64?")
				if enc, ok2 := strings.CutSuffix(enc, "?="); ok && ok2 {
					if dec, err := base64.StdEncoding.DecodeString(enc); err == nil {
						values[i] = string(dec)
					}
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}

func main() {
	dump := flag.Bool("dump", false, "Dump available tools as JSON and exit")
	v := flag.Bool("version", false, "Print version and exit")
	addr := flag.String("addr", "", "Listen address for HTTP (Streamable HTTP on /mcp, SSE on /sse), e.g. '127.0.0.1:8080' for local only or ':8080' for all interfaces. If empty, uses stdio.")
	grpcAddr := flag.String("grpc-addr", ":9590", "Listen address for gRPC service (e.g. '127.0.0.1:9590' for local only, or ':9590' for all interfaces)")
	mcpLimit := flag.Int("mcp-list-limit", 100, "Max artifacts to return in MCP list_artifacts")

	defaultDataDir := ".artifacts"
	if home, err := os.UserHomeDir(); err == nil {
		defaultDataDir = filepath.Join(home, "mlcartifact", "storage")
	}
	dataDir := flag.String("data-dir", defaultDataDir, "Base directory for artifact storage")
	flag.Parse()

	if *v {
		fmt.Printf("%s version: %s\n", name, version)
		return
	}

	// Initialize store and set in handlers
	store := storage.NewStore(*dataDir)
	slog.Info("initializing artifact store", "dir", *dataDir)
	artifactmcp.SetStore(store)
	artifactmcp.SetMCPListLimit(*mcpLimit)

	mcpServer := newServer()

	if *dump {
		if err := dumpTools(context.Background(), mcpServer); err != nil {
			slog.Error("dump failed", "err", err)
			os.Exit(1)
		}
		return
	}

	// Start Connect/gRPC server in background
	go func() {
		mux := http.NewServeMux()
		path, handler := protoconnect.NewArtifactServiceHandler(grpc.NewConnectServer(store))
		mux.Handle(path, handler)

		slog.Info("Connect/gRPC server started", "addr", *grpcAddr)

		// Setup CORS for browser access
		c := cors.New(cors.Options{
			AllowedOrigins: []string{"*"}, // Adjust in production
			AllowedMethods: []string{"GET", "POST", "OPTIONS"},
			AllowedHeaders: []string{"Connect-Protocol-Version", "Content-Type", "Accept", "Connect-Timeout-Ms", "X-User-Id"},
			ExposedHeaders: []string{"Content-Length"},
		})

		// We use h2c to support gRPC/HTTP2 without TLS
		if err := http.ListenAndServe(*grpcAddr, c.Handler(h2c.NewHandler(mux, &http2.Server{}))); err != nil {
			slog.Error("Connect/gRPC server failed", "err", err)
		}
	}()

	if *addr != "" {
		// HTTP mode: Streamable HTTP on /mcp, legacy SSE on /sse for older clients.
		getServer := func(*http.Request) *mcp.Server { return mcpServer }
		// Stateless: the SDK serves protocol 2026-07-28 over HTTP only in this mode.
		streamable := mcp.NewStreamableHTTPHandler(getServer, &mcp.StreamableHTTPOptions{Stateless: true})
		sse := mcp.NewSSEHandler(getServer, nil)
		// Reject foreign browser origins (DNS rebinding protection).
		cop := http.NewCrossOriginProtection()
		mux := http.NewServeMux()
		mux.Handle("/mcp", cop.Handler(decodeBase64Headers(streamable)))
		mux.Handle("/sse", cop.Handler(sse))
		slog.Info("HTTP server started", "addr", *addr, "streamable", "/mcp", "sse", "/sse", "name", name)
		if err := http.ListenAndServe(*addr, mux); err != nil {
			slog.Error("http server failed", "err", err)
			os.Exit(1)
		}
	} else {
		slog.Info("stdio server started", "name", name, "version", version)
		if err := mcpServer.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
			slog.Error("fatal error", "err", err)
			os.Exit(1)
		}
	}
}
