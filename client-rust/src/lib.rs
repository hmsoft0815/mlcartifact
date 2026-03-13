pub mod gen {
    tonic::include_proto!("artifact.v1");
}

use gen::artifact_service_client::ArtifactServiceClient;
use tonic::transport::Channel;

pub struct ArtifactClient {
    client: ArtifactServiceClient<Channel>,
}

impl ArtifactClient {
    pub async fn connect(dst: String) -> Result<Self, tonic::transport::Error> {
        let client = ArtifactServiceClient::connect(dst).await?;
        Ok(Self { client })
    }

    pub async fn write(&self, request: gen::WriteRequest) -> Result<gen::WriteResponse, tonic::Status> {
        let mut client = self.client.clone();
        let response = client.write(request).await?;
        Ok(response.into_inner())
    }

    pub async fn read(&self, id: String, user_id: Option<String>) -> Result<gen::ReadResponse, tonic::Status> {
        let mut client = self.client.clone();
        let request = gen::ReadRequest {
            id,
            user_id: user_id.unwrap_or_default(),
        };
        let response = client.read(request).await?;
        Ok(response.into_inner())
    }

    pub async fn list(&self, user_id: Option<String>, limit: Option<i32>, offset: Option<i32>) -> Result<gen::ListResponse, tonic::Status> {
        let mut client = self.client.clone();
        let request = gen::ListRequest {
            user_id: user_id.unwrap_or_default(),
            limit: limit.unwrap_or_default(),
            offset: offset.unwrap_or_default(),
            source: String::new(),
        };
        let response = client.list(request).await?;
        Ok(response.into_inner())
    }

    pub async fn delete(&self, id: String, user_id: Option<String>) -> Result<gen::DeleteResponse, tonic::Status> {
        let mut client = self.client.clone();
        let request = gen::DeleteRequest {
            id,
            user_id: user_id.unwrap_or_default(),
        };
        let response = client.delete(request).await?;
        Ok(response.into_inner())
    }
}
