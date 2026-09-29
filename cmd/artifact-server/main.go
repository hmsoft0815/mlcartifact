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
	"time"

	"github.com/hmsoft0815/mlcartifact/internal/auth"
	"github.com/hmsoft0815/mlcartifact/internal/grpc"
	artifactmcp "github.com/hmsoft0815/mlcartifact/internal/mcp"
	"github.com/hmsoft0815/mlcartifact/internal/storage"
	"github.com/hmsoft0815/mlcartifact/proto/protoconnect"
	"github.com/rs/cors"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"connectrpc.com/connect"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var (
	version = "0.5.6"
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
	defer func() { _ = cs.Close() }()
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

const maxRequestBytes = 64 << 20

func limitBody(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxRequestBytes)
		next.ServeHTTP(w, r)
	})
}

func main() {
	defaultGrpcAddr := os.Getenv("ARTIFACT_GRPC_ADDR")
	if defaultGrpcAddr == "" {
		defaultGrpcAddr = "127.0.0.1:9590"
	}
	defaultGrpcToken := os.Getenv("ARTIFACT_GRPC_TOKEN")
	if defaultGrpcToken == "" {
		defaultGrpcToken = os.Getenv("ARTIFACT_TOKEN")
	}
	defaultRequireLocalhost := false
	if envVal := os.Getenv("ARTIFACT_REQUIRE_TOKEN_LOCALHOST"); envVal == "true" || envVal == "1" {
		defaultRequireLocalhost = true
	}
	defaultAuthEndpoint := os.Getenv("ARTIFACT_AUTH_ENDPOINT")
	defaultAuthHeaders := os.Getenv("ARTIFACT_AUTH_HEADERS")
	defaultAuthServiceName := os.Getenv("ARTIFACT_AUTH_SERVICE_NAME")
	if defaultAuthServiceName == "" {
		defaultAuthServiceName = "mlcartifact"
	}
	defaultAuthCacheTTL := os.Getenv("ARTIFACT_AUTH_CACHE_TTL")
	if defaultAuthCacheTTL == "" {
		defaultAuthCacheTTL = "60s"
	}

	dump := flag.Bool("dump", false, "Dump available tools as JSON and exit")
	v := flag.Bool("version", false, "Print version and exit")
	addr := flag.String("addr", "", "Listen address for HTTP (Streamable HTTP on /mcp, SSE on /sse), e.g. '127.0.0.1:8080' for local only or '0.0.0.0:8080' for all interfaces. If empty, uses stdio.")
	grpcAddr := flag.String("grpc-addr", defaultGrpcAddr, "Listen address for gRPC service (default: '127.0.0.1:9590', or '0.0.0.0:9590' for all interfaces)")
	grpcToken := flag.String("grpc-token", defaultGrpcToken, "Authentication token required for remote (non-loopback) access. Can also be set via ARTIFACT_GRPC_TOKEN.")
	requireTokenLocalhost := flag.Bool("require-token-localhost", defaultRequireLocalhost, "Require authentication token even for localhost / loopback connections (default: false)")
	authEndpoint := flag.String("auth-endpoint", defaultAuthEndpoint, "Optional HTTP Forward-Auth or OAuth2 UserInfo endpoint URL (e.g. mlcauth or Supabase) to validate tokens")
	authHeaders := flag.String("auth-header", defaultAuthHeaders, "Optional comma-separated extra headers for auth-endpoint (e.g. 'apikey:xyz')")
	authServiceName := flag.String("auth-service-name", defaultAuthServiceName, "Optional service name required in user claims/services (default: 'mlcartifact')")
	authCacheTTL := flag.String("auth-cache-ttl", defaultAuthCacheTTL, "Cache duration for validated tokens (default: '60s')")
	corsOrigins := flag.String("cors-origins", os.Getenv("ARTIFACT_CORS_ORIGINS"), "Comma-separated list of allowed CORS browser origins (default: none / cross-origin denied)")
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

	// Build TokenValidator (HTTP Forward-Auth / Supabase or static token)
	var validator auth.TokenValidator
	if *authEndpoint != "" {
		var extraHeaders map[string]string
		if *authHeaders != "" {
			extraHeaders = make(map[string]string)
			for _, part := range strings.Split(*authHeaders, ",") {
				part = strings.TrimSpace(part)
				if k, v, ok := strings.Cut(part, ":"); ok {
					extraHeaders[strings.TrimSpace(k)] = strings.TrimSpace(v)
				}
			}
		}
		cacheDuration, err := time.ParseDuration(*authCacheTTL)
		if err != nil {
			slog.Error("invalid -auth-cache-ttl", "err", err)
			os.Exit(1)
		}
		validator = auth.NewHTTPTokenValidator(auth.HTTPValidatorConfig{
			Endpoint:     *authEndpoint,
			ExtraHeaders: extraHeaders,
			ServiceName:  *authServiceName,
			CacheTTL:     cacheDuration,
			Client:       &http.Client{Timeout: 5 * time.Second},
		})
		slog.Info("configured HTTP token validator",
			"endpoint", *authEndpoint,
			"service_name", *authServiceName,
			"cache_ttl", cacheDuration,
		)
	} else if *grpcToken != "" {
		validator = auth.NewStaticTokenValidator(*grpcToken)
	}
	hasAuth := validator != nil

	// Parse allowed CORS origins
	var allowedOrigins []string
	if *corsOrigins != "" {
		for _, o := range strings.Split(*corsOrigins, ",") {
			o = strings.TrimSpace(o)
			if o != "" {
				allowedOrigins = append(allowedOrigins, o)
			}
		}
	}

	// Validate Connect/gRPC address
	resolvedGrpcAddr, isGrpcLoopback, err := grpc.ListenAddr(*grpcAddr)
	if err != nil {
		slog.Error("invalid -grpc-addr", "err", err)
		os.Exit(1)
	}
	if !isGrpcLoopback && !hasAuth {
		slog.Error("-grpc-addr is not restricted to loopback, but no authentication is configured (-grpc-token or -auth-endpoint)")
		os.Exit(1)
	}

	// Validate HTTP address if enabled
	var resolvedAddr string
	var isAddrLoopback bool
	if *addr != "" {
		resolvedAddr, isAddrLoopback, err = grpc.ListenAddr(*addr)
		if err != nil {
			slog.Error("invalid -addr", "err", err)
			os.Exit(1)
		}
		if !isAddrLoopback && !hasAuth {
			slog.Error("-addr is not restricted to loopback, but no authentication is configured (-grpc-token or -auth-endpoint)")
			os.Exit(1)
		}
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
		path, handler := protoconnect.NewArtifactServiceHandler(
			grpc.NewConnectServer(store),
			connect.WithInterceptors(grpc.NewAuthInterceptorWithValidator(
				validator,
				grpc.WithRequireTokenLocalhost(*requireTokenLocalhost),
				grpc.WithTrustedOrigins(allowedOrigins),
			)),
		)

		// Reject unauthorized cross-origin browser requests (CSRF / DNS-rebinding protection)
		cop := http.NewCrossOriginProtection()
		for _, o := range allowedOrigins {
			if err := cop.AddTrustedOrigin(o); err != nil {
				slog.Warn("invalid cors origin", "origin", o, "err", err)
			}
		}
		mux.Handle(path, cop.Handler(handler))

		slog.Info("Connect/gRPC server started",
			"addr", resolvedGrpcAddr,
			"loopback", isGrpcLoopback,
			"remote_token_protected", hasAuth,
			"require_token_localhost", *requireTokenLocalhost,
			"allowed_cors_origins", allowedOrigins,
		)

		var serverHandler http.Handler = mux
		if len(allowedOrigins) > 0 {
			// Setup CORS only for explicitly allowed browser origins
			c := cors.New(cors.Options{
				AllowedOrigins: allowedOrigins,
				AllowedMethods: []string{"GET", "POST", "OPTIONS"},
				AllowedHeaders: []string{
					"Connect-Protocol-Version",
					"Content-Type",
					"Accept",
					"Connect-Timeout-Ms",
					"X-User-Id",
					"Authorization",
					"X-Artifact-Token",
				},
				ExposedHeaders: []string{"Content-Length"},
			})
			serverHandler = c.Handler(serverHandler)
		}

		// We use h2c to support gRPC/HTTP2 without TLS
		if err := http.ListenAndServe(resolvedGrpcAddr, h2c.NewHandler(serverHandler, &http2.Server{})); err != nil {
			slog.Error("Connect/gRPC server failed", "err", err)
		}
	}()

	if *addr != "" {
		// HTTP mode: Streamable HTTP on /mcp, legacy SSE on /sse for older clients.
		getServer := func(*http.Request) *mcp.Server { return mcpServer }
		// Stateless: the SDK serves protocol 2026-07-28 over HTTP only in this mode.
		streamable := mcp.NewStreamableHTTPHandler(getServer, &mcp.StreamableHTTPOptions{Stateless: true})
		sse := mcp.NewSSEHandler(getServer, nil)

		mcpAuth := grpc.HTTPAuthMiddlewareWithValidator(validator, *requireTokenLocalhost, allowedOrigins)

		// Reject foreign browser origins (DNS rebinding protection).
		cop := http.NewCrossOriginProtection()
		for _, o := range allowedOrigins {
			if err := cop.AddTrustedOrigin(o); err != nil {
				slog.Warn("invalid cors origin", "origin", o, "err", err)
			}
		}

		mux := http.NewServeMux()
		mux.Handle("/mcp", cop.Handler(mcpAuth(decodeBase64Headers(limitBody(streamable)))))
		mux.Handle("/sse", cop.Handler(mcpAuth(limitBody(sse))))

		if hasAuth {
			// Protected Resource Metadata (RFC 9728) for MCP authentication discovery
			prm := func(w http.ResponseWriter, r *http.Request) {
				scheme := "http"
				if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
					scheme = "https"
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = fmt.Fprintf(w, `{"resource":"%s://%s/mcp","bearer_methods_supported":["header"],"resource_name":"mlcartifact MCP","resource_documentation":"https://mlcgo.eu/products/mlcartifact/"}`,
					scheme, r.Host)
			}
			mux.HandleFunc("/.well-known/oauth-protected-resource", prm)
			mux.HandleFunc("/.well-known/oauth-protected-resource/mcp", prm)
		}

		mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			_, _ = fmt.Fprintf(w, "ok %s v%s\n", name, version)
		})

		slog.Info("HTTP server started",
			"addr", resolvedAddr,
			"loopback", isAddrLoopback,
			"streamable", "/mcp",
			"sse", "/sse",
			"name", name,
		)
		if err := http.ListenAndServe(resolvedAddr, mux); err != nil {
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
