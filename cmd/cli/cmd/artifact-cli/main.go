// Copyright (c) 2026 Michael Lechner. All rights reserved.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hmsoft0815/mlcartifact"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	addr := flag.String("addr", os.Getenv("ARTIFACT_GRPC_ADDR"), "Artifact server gRPC address")
	if *addr == "" {
		*addr = "localhost:50051"
	}

	flag.Parse()

	if flag.NArg() < 1 {
		usage()
		os.Exit(1)
	}

	cli, err := mlcartifact.NewClientWithAddr(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer cli.Close()

	cmd := flag.Arg(0)
	switch cmd {
	case "list":
		handleList(cli, flag.Args()[1:])
	case "delete":
		handleDelete(cli, flag.Args()[1:])
	case "create":
		handleCreate(cli, flag.Args()[1:])
	case "download":
		handleDownload(cli, flag.Args()[1:])
	default:
		fmt.Printf("Unknown command: %s\n", cmd)
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Println("Usage: artifact-cli [options] <command> [args]")
	fmt.Println("Options:")
	fmt.Println("  -addr string  gRPC address (default: ARTIFACT_GRPC_ADDR or localhost:50051)")
	fmt.Println("Commands:")
	fmt.Println("  list [--limit N] [--offset M] [--user ID]")
	fmt.Println("  delete <id> [--user ID]")
	fmt.Println("  create <file> [--name NAME] [--description DESC] [--user ID] [--expires HOURS]")
	fmt.Println("  download <id/filename> <local-path> [--user ID]")
}

func handleList(cli *mlcartifact.Client, args []string) {
	fs := flag.NewFlagSet("list", flag.ExitOnError)
	limit := fs.Int("limit", 0, "Limit items")
	offset := fs.Int("offset", 0, "Offset items")
	user := fs.String("user", "", "Filter by user ID")
	fs.Parse(args)

	// Since mlcartifact.Client doesn't directly expose limit/offset yet,
	// we will need to update it or use the proto client directly.
	// For now, let's assume we update the client.

	// Actually, I'll update mlcartifact.Client.List signature to support pagination or add options.
	// But to keep it simple and compileable now, let's just use raw List and implement pagination in client if needed.
	// WAit, if I updated the proto, I should update the client too.

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	items, err := cli.List(ctx, *user,
		mlcartifact.WithLimit(int32(*limit)),
		mlcartifact.WithOffset(int32(*offset)),
	)
	if err != nil {
		log.Fatalf("List failed: %v", err)
	}

	fmt.Printf("%-10s %-30s %-20s %-10s %-20s\n", "ID", "Filename", "Mime", "Size", "Created")
	fmt.Println(strings.Repeat("-", 100))
	for _, item := range items.Items {
		fmt.Printf("%-10s %-30s %-20s %-10d %-20s\n", item.Id, item.Filename, item.MimeType, item.SizeBytes, item.CreatedAt)
	}
}

func handleDelete(cli *mlcartifact.Client, args []string) {
	fs := flag.NewFlagSet("delete", flag.ExitOnError)
	user := fs.String("user", "", "Scope to user ID")
	fs.Parse(args)

	if fs.NArg() < 1 {
		log.Fatal("Artifact ID required")
	}
	id := fs.Arg(0)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := cli.Delete(ctx, id, mlcartifact.WithDeleteUserID(*user))
	if err != nil {
		log.Fatalf("Delete failed: %v", err)
	}
	fmt.Println("Successfully deleted artifact:", id)
}

func handleCreate(cli *mlcartifact.Client, args []string) {
	fs := flag.NewFlagSet("create", flag.ExitOnError)
	name := fs.String("name", "", "Override filename")
	desc := fs.String("description", "", "Add description")
	user := fs.String("user", os.Getenv("ARTIFACT_USER_ID"), "User ID")
	expires := fs.Int("expires", 24, "Expiration in hours")
	fs.Parse(args)

	if fs.NArg() < 1 {
		log.Fatal("Local file path required")
	}
	path := fs.Arg(0)

	data, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("Failed to read file: %v", err)
	}

	filename := *name
	if filename == "" {
		filename = filepath.Base(path)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	res, err := cli.Write(ctx, filename, data,
		mlcartifact.WithUserID(*user),
		mlcartifact.WithDescription(*desc),
		mlcartifact.WithExpiresHours(int32(*expires)),
		mlcartifact.WithSource("artifact-cli"),
	)
	if err != nil {
		log.Fatalf("Create failed: %v", err)
	}

	fmt.Printf("Artifact created successfully!\nID: %s\nURI: %s\n", res.Id, res.Uri)
}

func handleDownload(cli *mlcartifact.Client, args []string) {
	fs := flag.NewFlagSet("download", flag.ExitOnError)
	user := fs.String("user", "", "Scope to user ID")
	fs.Parse(args)

	if fs.NArg() < 2 {
		log.Fatal("Artifact ID and local path required")
	}
	id := fs.Arg(0)
	dest := fs.Arg(1)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	res, err := cli.Read(ctx, id, mlcartifact.WithReadUserID(*user))
	if err != nil {
		log.Fatalf("Read failed: %v", err)
	}

	if err := os.WriteFile(dest, res.Content, 0644); err != nil {
		log.Fatalf("Failed to write to file: %v", err)
	}

	fmt.Printf("Successfully downloaded %s (%s) to %s\n", res.Filename, res.MimeType, dest)
}
