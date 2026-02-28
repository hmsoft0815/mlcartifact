// Copyright (c) 2026 Michael Lechner. All rights reserved.
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
)

type JSONRPCRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

type JSONRPCResponse struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Result  any    `json:"result,omitempty"`
	Error   any    `json:"error,omitempty"`
}

func main() {
	fmt.Println("🚀 artifact-server integration test")

	// 1. Build
	_ = exec.Command("go", "build", "-o", "artifact-server", "./main.go").Run()
	defer os.Remove("artifact-server")

	// 2. Start
	cmd := exec.Command("./artifact-server")
	stdin, _ := cmd.StdinPipe()
	stdout, _ := cmd.StdoutPipe()
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		fmt.Printf("FAILED to start server: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = cmd.Process.Kill() }()

	reader := bufio.NewReader(stdout)
	id := 1

	runTest := func(name string, args map[string]any) {
		fmt.Printf("\n--- TEST: %s ---\n", name)
		sendRequest(stdin, "tools/call", map[string]any{"name": "write_artifact", "arguments": args}, id)
		id++

		line, _ := reader.ReadBytes('\n')
		var resp JSONRPCResponse
		if err := json.Unmarshal(line, &resp); err != nil {
			fmt.Printf("❌ JSON Unmarshal Error: %v\n", err)
			return
		}

		if resp.Error != nil {
			fmt.Printf("❌ Error: %v\n", resp.Error)
			return
		}

		resultMap, _ := resp.Result.(map[string]any)
		contentList, _ := resultMap["content"].([]any)
		fmt.Printf("📦 Received %d content blocks\n", len(contentList))
		for _, c := range contentList {
			fmt.Printf("   ↳ %s\n", c.(map[string]any)["text"].(string))
		}
	}

	// Case 1: Simple artifact
	runTest("Simple Artifact", map[string]any{
		"filename": "hello.txt",
		"content":  "Hello World from Artifact Server!",
	})

		// Case 2: Multi-hour expiration
		runTest("Expiring Artifact", map[string]any{
			"filename":          "short_lived.log",
			"content":           "This will disappear soon.",
			"expires_in_hours": 1,
		})
	
		// Case 3: Explicit MIME type
		runTest("Explicit MIME", map[string]any{
			"filename":  "data.csv",
			"content":   "a,b,c\n1,2,3",
			"mime_type": "text/csv",
		})
	
		// Case 4: Auto Detection
		runTest("Auto Detection", map[string]any{
			"filename": "document.md",
			"content":  "# Heading\nContent",
		})
	
		fmt.Println("\n🏁 All tests completed.")
	
}

func sendRequest(w io.Writer, method string, params any, id int) {
	req := JSONRPCRequest{JSONRPC: "2.0", ID: id, Method: method, Params: params}
	b, _ := json.Marshal(req)
	_, _ = w.Write(append(b, '\n'))
}
