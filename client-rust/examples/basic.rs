use mlcartifact::{ArtifactClient, WriteOptions};

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    println!("--- mlcartifact Rust 'Hello World' Example ---");

    // Connects using ARTIFACT_GRPC_ADDR (defaulting to http://localhost:9590),
    // and respects ARTIFACT_GRPC_TOKEN / ARTIFACT_USER_ID if set.
    let client = ArtifactClient::from_env().await?;

    // 1. Write 3 artifacts
    let items = [
        ("artifact1.txt", b"Content for artifact A"),
        ("artifact2.txt", b"Content for artifact B"),
        ("artifact3.txt", b"Content for artifact C"),
    ];

    let mut ids = Vec::new();
    for (name, content) in &items {
        let response = client
            .write_file(
                *name,
                content.to_vec(),
                WriteOptions {
                    mime_type: Some("text/plain".into()),
                    expires_hours: Some(1),
                    source: Some("rust-example".into()),
                    description: Some(format!("Created by Rust example: {}", name)),
                    ..Default::default()
                },
            )
            .await?;
        println!("Wrote: {} (ID: {})", name, response.id);
        ids.push(response.id);
    }

    // 2. Delete one (artifact 2)
    println!("Deleting artifact 2 (ID: {})...", ids[1]);
    client.delete(ids[1].clone(), None).await?;

    // 3. Retrieve others and compare
    for i in [0, 2] {
        let read_res = client.read(ids[i].clone(), None).await?;
        if read_res.content != items[i].1 {
            panic!(
                "Content mismatch for {}! Expected '{:?}', got '{:?}'",
                items[i].0, items[i].1, read_res.content
            );
        }
        println!("Verified: {} (ID: {}) content matches.", items[i].0, ids[i]);
    }

    // 4. Verify artifact 2 is gone
    match client.read(ids[1].clone(), None).await {
        Err(status) if status.code() == tonic::Code::NotFound => {
            println!("Verified: Artifact 2 is indeed gone.");
        }
        Ok(_) => panic!("Error: Artifact 2 should have been deleted but was found!"),
        Err(e) => panic!("Error during verify: {:?}", e),
    }

    // 5. Demonstrate VFS, patch, and find
    let run_id = std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)?
        .as_nanos();
    let vfs_base = format!("/rust-example-{}", run_id);

    client
        .write_file(
            "readme.md",
            b"line 1\nline 2\nline 3".to_vec(),
            WriteOptions {
                virtual_path: Some(format!("{}/docs/readme.md", vfs_base)),
                ..Default::default()
            },
        )
        .await?;

    client
        .write_file(
            "notes.txt",
            b"meeting notes".to_vec(),
            WriteOptions {
                virtual_path: Some(format!("{}/notes.txt", vfs_base)),
                ..Default::default()
            },
        )
        .await?;

    // Patch line 2 of readme.md
    let patch_res = client
        .patch(
            format!("{}/docs/readme.md", vfs_base),
            b"LINE 2".to_vec(),
            Some(2),
            Some(2),
            false,
            None,
        )
        .await?;
    println!("Patched {}/docs/readme.md (success={})", vfs_base, patch_res.success);

    // Find files matching pattern
    let find_res = client
        .find(format!("{}/*/*.md", vfs_base), None)
        .await?;
    let found_paths: Vec<_> = find_res.items.iter().map(|a| a.virtual_path.as_str()).collect();
    println!("Find {}/*/*.md -> {:?}", vfs_base, found_paths);

    println!("--- Example finished successfully ---");

    Ok(())
}
