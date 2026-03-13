package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hmsoft0815/mlcartifact"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 1. Initialize client
	client, err := mlcartifact.NewClient()
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	fmt.Println("--- mlcartifact Go 'Hello World' Example ---")

	// 2. Write 3 artifacts
	items := []struct {
		name    string
		content string
	}{
		{"artifact1.txt", "Content for artifact A"},
		{"artifact2.txt", "Content for artifact B"},
		{"artifact3.txt", "Content for artifact C"},
	}

	ids := make([]string, len(items))
	for i, item := range items {
		res, err := client.Write(ctx, item.name, []byte(item.content))
		if err != nil {
			log.Fatalf("Failed to write %s: %v", item.name, err)
		}
		ids[i] = res.Id
		fmt.Printf("Wrote: %s (ID: %s)\n", item.name, res.Id)
	}

	// 3. Delete one (artifact 2)
	fmt.Printf("Deleting artifact 2 (ID: %s)...\n", ids[1])
	_, err = client.Delete(ctx, ids[1])
	if err != nil {
		log.Fatalf("Failed to delete: %v", err)
	}

	// 4. Retrieve others and compare
	toCheck := []int{0, 2}
	for _, idx := range toCheck {
		res, err := client.Read(ctx, ids[idx])
		if err != nil {
			log.Fatalf("Failed to read %s: %v", items[idx].name, err)
		}

		if !bytes.Equal(res.Content, []byte(items[idx].content)) {
			log.Fatalf("Content mismatch for %s! Expected '%s', got '%s'", 
				items[idx].name, items[idx].content, string(res.Content))
		}
		fmt.Printf("Verified: %s (ID: %s) content matches.\n", items[idx].name, ids[idx])
	}

	// 5. Verify artifact 2 is gone
	_, err = client.Read(ctx, ids[1])
	if err == nil {
		log.Fatal("Error: Artifact 2 should have been deleted but was found!")
	}
	fmt.Println("Verified: Artifact 2 is indeed gone.")

	fmt.Println("--- Example finished successfully ---")
}
