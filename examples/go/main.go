package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"path"
	"time"

	"github.com/hmsoft0815/mlcartifact/client"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 1. Initialize client
	c, err := client.NewClient()
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer c.Close()

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
		res, err := c.Write(ctx, item.name, []byte(item.content))
		if err != nil {
			log.Fatalf("Failed to write %s: %v", item.name, err)
		}
		ids[i] = res.Id
		fmt.Printf("Wrote: %s (ID: %s)\n", item.name, res.Id)
	}

	// 3. Delete one (artifact 2)
	fmt.Printf("Deleting artifact 2 (ID: %s)...\n", ids[1])
	_, err = c.Delete(ctx, ids[1])
	if err != nil {
		log.Fatalf("Failed to delete: %v", err)
	}

	// 4. Retrieve others and compare
	toCheck := []int{0, 2}
	for _, idx := range toCheck {
		res, err := c.Read(ctx, ids[idx])
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
	_, err = c.Read(ctx, ids[1])
	if err == nil {
		log.Fatal("Error: Artifact 2 should have been deleted but was found!")
	}
	fmt.Println("Verified: Artifact 2 is indeed gone.")

	// 6. Virtual file system: store by path, patch in place, list and find
	vpath := fmt.Sprintf("/go-example-%d/docs/readme.md", time.Now().UnixNano())
	if _, err := c.Write(ctx, "readme.md", []byte("line 1\nline 2\nline 3"), client.WithVirtualPath(vpath)); err != nil {
		log.Fatalf("Failed to write %s: %v", vpath, err)
	}
	// Lines are 0-based, end exclusive: (1, 2) replaces "line 2"
	if _, err := c.Patch(ctx, vpath, []byte("LINE 2"), client.WithLines(1, 2)); err != nil {
		log.Fatalf("Failed to patch: %v", err)
	}
	res, err := c.Read(ctx, vpath)
	if err != nil || string(res.Content) != "line 1\nLINE 2\nline 3" {
		log.Fatalf("Patch result wrong: %q (%v)", res.GetContent(), err)
	}
	fmt.Printf("Patched %s: %q\n", vpath, res.Content)

	dir := path.Dir(path.Dir(vpath))
	listing, err := c.List(ctx, "", client.WithDirPath(dir))
	if err != nil || len(listing.Items) != 1 || !listing.Items[0].IsDirectory {
		log.Fatalf("Expected one directory in %s: %v (%v)", dir, listing.GetItems(), err)
	}
	fmt.Printf("List %s: %s/ (directory)\n", dir, listing.Items[0].Filename)

	found, err := c.Find(ctx, dir+"/*/*.md")
	if err != nil || len(found.Items) != 1 {
		log.Fatalf("Find failed: %v (%v)", found.GetItems(), err)
	}
	fmt.Printf("Find %s/*/*.md -> %s\n", dir, found.Items[0].VirtualPath)

	if _, err := c.Delete(ctx, vpath); err != nil {
		log.Fatalf("Failed to delete %s: %v", vpath, err)
	}

	fmt.Println("--- Example finished successfully ---")
}
