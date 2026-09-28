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

	"github.com/hmsoft0815/mlcartifact/client"
)

var version = "dev"

func main() {
	defaultAddr := os.Getenv("ARTIFACT_GRPC_ADDR")
	if defaultAddr == "" {
		defaultAddr = "localhost:9590"
	}
	defaultToken := os.Getenv("ARTIFACT_GRPC_TOKEN")
	if defaultToken == "" {
		defaultToken = os.Getenv("ARTIFACT_TOKEN")
	}
	defaultUser := os.Getenv("ARTIFACT_USER_ID")

	addr := flag.String("addr", defaultAddr, "Artifact server gRPC address")
	token := flag.String("token", defaultToken, "Authentication token for remote server (default: ARTIFACT_GRPC_TOKEN or ARTIFACT_TOKEN)")
	user := flag.String("user", defaultUser, "User ID scoping (default: ARTIFACT_USER_ID)")
	v := flag.Bool("version", false, "Print version and exit")

	flag.Usage = usage
	flag.Parse()

	if *v {
		fmt.Printf("artifact-cli version: %s\n", version)
		return
	}

	if flag.NArg() < 1 {
		usage()
		os.Exit(1)
	}

	cli, err := client.NewClientWithAddr(*addr, client.WithToken(*token))
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer cli.Close()

	cmd := flag.Arg(0)
	switch cmd {
	case "list":
		handleList(cli, flag.Args()[1:], *user)
	case "delete":
		handleDelete(cli, flag.Args()[1:], *user)
	case "create":
		handleCreate(cli, flag.Args()[1:], *user)
	case "download":
		handleDownload(cli, flag.Args()[1:], *user)
	default:
		fmt.Printf("Unknown command: %s\n", cmd)
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Println("Usage: artifact-cli [global options] <command> [command options] [args]")
	fmt.Println()
	fmt.Println("Global Options:")
	fmt.Println("  -addr string   gRPC address (default: ARTIFACT_GRPC_ADDR or localhost:9590)")
	fmt.Println("  -token string  Authentication token for remote server (default: ARTIFACT_GRPC_TOKEN)")
	fmt.Println("  -user string   Default user ID for operations (default: ARTIFACT_USER_ID)")
	fmt.Println("  -version       Print version and exit")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  create <file> [--name NAME] [--description DESC] [--user ID] [--expires HOURS] [-q]")
	fmt.Println("      Uploads a file to the artifact store.")
	fmt.Println("      -q: Quiet mode, prints only the artifact ID (useful for scripts: ID=$(artifact-cli create -q datei))")
	fmt.Println()
	fmt.Println("  download <id/filename> <local-path> [--user ID]")
	fmt.Println("      Downloads an artifact to a local file. Inherits ARTIFACT_USER_ID or global -user.")
	fmt.Println()
	fmt.Println("  list [--limit N] [--offset M] [--user ID]")
	fmt.Println("      Lists artifacts. Inherits ARTIFACT_USER_ID or global -user.")
	fmt.Println()
	fmt.Println("  delete <id> [--user ID]")
	fmt.Println("      Deletes an artifact permanently. Inherits ARTIFACT_USER_ID or global -user.")
}

func handleList(cli *client.Client, args []string, defaultUser string) {
	fs := flag.NewFlagSet("list", flag.ExitOnError)
	limit := fs.Int("limit", 0, "Limit items")
	offset := fs.Int("offset", 0, "Offset items")
	user := fs.String("user", defaultUser, "Filter by user ID (default: ARTIFACT_USER_ID)")
	if err := fs.Parse(args); err != nil {
		log.Fatalf("Parse error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	items, err := cli.List(ctx, *user,
		client.WithLimit(int32(*limit)),
		client.WithOffset(int32(*offset)),
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

func handleDelete(cli *client.Client, args []string, defaultUser string) {
	fs := flag.NewFlagSet("delete", flag.ExitOnError)
	user := fs.String("user", defaultUser, "Scope to user ID (default: ARTIFACT_USER_ID)")
	if err := fs.Parse(args); err != nil {
		log.Fatalf("Parse error: %v", err)
	}

	if fs.NArg() < 1 {
		log.Fatal("Artifact ID required")
	}
	id := fs.Arg(0)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var opts []client.DeleteOption
	if *user != "" {
		opts = append(opts, client.WithDeleteUserID(*user))
	}

	_, err := cli.Delete(ctx, id, opts...)
	if err != nil {
		log.Fatalf("Delete failed: %v", err)
	}
	fmt.Println("Successfully deleted artifact:", id)
}

func handleCreate(cli *client.Client, args []string, defaultUser string) {
	fs := flag.NewFlagSet("create", flag.ExitOnError)
	name := fs.String("name", "", "Override filename")
	desc := fs.String("description", "", "Add description")
	user := fs.String("user", defaultUser, "User ID (default: ARTIFACT_USER_ID)")
	expires := fs.Int("expires", 24, "Expiration in hours")
	quiet := fs.Bool("q", false, "Quiet mode: print only the artifact ID")
	if err := fs.Parse(args); err != nil {
		log.Fatalf("Parse error: %v", err)
	}

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
		client.WithUserID(*user),
		client.WithDescription(*desc),
		client.WithExpiresHours(int32(*expires)),
		client.WithSource("artifact-cli"),
	)
	if err != nil {
		log.Fatalf("Create failed: %v", err)
	}

	if *quiet {
		fmt.Println(res.Id)
	} else {
		fmt.Printf("Artifact created successfully!\nID: %s\nURI: %s\n", res.Id, res.Uri)
	}
}

func handleDownload(cli *client.Client, args []string, defaultUser string) {
	fs := flag.NewFlagSet("download", flag.ExitOnError)
	user := fs.String("user", defaultUser, "Scope to user ID (default: ARTIFACT_USER_ID)")
	if err := fs.Parse(args); err != nil {
		log.Fatalf("Parse error: %v", err)
	}

	if fs.NArg() < 2 {
		log.Fatal("Artifact ID and local path required")
	}
	id := fs.Arg(0)
	dest := fs.Arg(1)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var opts []client.ReadOption
	if *user != "" {
		opts = append(opts, client.WithReadUserID(*user))
	}

	res, err := cli.Read(ctx, id, opts...)
	if err != nil {
		log.Fatalf("Read failed: %v", err)
	}

	if err := os.WriteFile(dest, res.Content, 0644); err != nil {
		log.Fatalf("Failed to write to file: %v", err)
	}

	fmt.Printf("Successfully downloaded %s (%s) to %s\n", res.Filename, res.MimeType, dest)
}
