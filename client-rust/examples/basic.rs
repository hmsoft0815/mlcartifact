use mlcartifact::gen::artifact_service_client::ArtifactServiceClient;
use mlcartifact::gen::{WriteRequest, ReadRequest, DeleteRequest};
use tonic::Request;
use std::collections::HashMap;

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let addr = "http://localhost:9590";
    println!("Connecting to {}...", addr);
    println!("--- mlcartifact Rust 'Hello World' Example ---");

    let mut client = ArtifactServiceClient::connect(addr).await?;

    // 1. Write 3 artifacts
    let items = vec![
        ("artifact1.txt", b"Content for artifact A"),
        ("artifact2.txt", b"Content for artifact B"),
        ("artifact3.txt", b"Content for artifact C"),
    ];

    let mut ids = Vec::new();
    for (name, content) in &items {
        let request = Request::new(WriteRequest {
            filename: name.to_string(),
            content: content.to_vec(),
            mime_type: "text/plain".into(),
            expires_hours: 1,
            source: "rust-example".into(),
            metadata: HashMap::new(),
            user_id: "".into(),
            description: format!("Created by Rust example: {}", name),
        });

        let response = client.write(request).await?.into_inner();
        println!("Wrote: {} (ID: {})", name, response.id);
        ids.push(response.id);
    }

    // 2. Delete one (artifact 2)
    println!("Deleting artifact 2 (ID: {})...", ids[1]);
    let delete_req = Request::new(DeleteRequest {
        id: ids[1].clone(),
        user_id: "".into(),
    });
    client.delete(delete_req).await?;

    // 3. Retrieve others and compare
    for i in [0, 2] {
        let read_req = Request::new(ReadRequest {
            id: ids[idx_to_usize(i)].clone(),
            user_id: "".into(),
        });
        let read_res = client.read(read_req).await?.into_inner();
        
        if read_res.content != items[idx_to_usize(i)].1 {
            panic!("Content mismatch for {}! Expected '{:?}', got '{:?}'", 
                   items[idx_to_usize(i)].0, items[idx_to_usize(i)].1, read_res.content);
        }
        println!("Verified: {} (ID: {}) content matches.", items[idx_to_usize(i)].0, ids[idx_to_usize(i)]);
    }

    // 4. Verify artifact 2 is gone
    let read_req = Request::new(ReadRequest {
        id: ids[1].clone(),
        user_id: "".into(),
    });
    let read_res = client.read(read_req).await;
    match read_res {
        Err(status) if status.code() == tonic::Code::NotFound => {
            println!("Verified: Artifact 2 is indeed gone.");
        }
        Ok(_) => panic!("Error: Artifact 2 should have been deleted but was found!"),
        Err(e) => panic!("Error during verify: {:?}", e),
    }

    println!("--- Example finished successfully ---");

    Ok(())
}

fn idx_to_usize(i: i32) -> usize {
    i as usize
}
