use mlcartifact::gen::artifact_service_client::ArtifactServiceClient;
use mlcartifact::gen::WriteRequest;
use tonic::Request;

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let addr = "http://localhost:9590";
    println!("Connecting to {}...", addr);

    let mut client = ArtifactServiceClient::connect(addr).await?;

    let request = Request::new(WriteRequest {
        filename: "rust-test.txt".into(),
        content: b"Hello from Rust!".to_vec(),
        mime_type: "text/plain".into(),
        expires_hours: 1,
        source: "rust-example".into(),
        metadata: std::collections::HashMap::new(),
        user_id: "".into(),
        description: "Created by Rust example".into(),
    });

    let response = client.write(request).await?;

    println!("Success! Artifact created: {:?}", response.into_inner());

    Ok(())
}
