//! Rust client for the mlcartifact service (gRPC, `artifact.v1`).
//!
//! The raw generated types and client live in [`gen`]; [`ArtifactClient`] is a
//! convenience wrapper providing token authentication, environment variable
//! defaults, and high-level methods.

pub mod gen {
    tonic::include_proto!("artifact.v1");
}

pub const VERSION: &str = env!("CARGO_PKG_VERSION");

use std::collections::HashMap;

use gen::artifact_service_client::ArtifactServiceClient;
use tonic::transport::Channel;

/// Optional parameters for [`ArtifactClient::write_file`].
#[derive(Debug, Clone, Default)]
pub struct WriteOptions {
    /// MIME type; auto-detected by the server from the filename if empty.
    pub mime_type: Option<String>,
    /// Expiry in hours; server default (24) if `None`.
    pub expires_hours: Option<i32>,
    /// Source server name for auditing.
    pub source: Option<String>,
    pub metadata: HashMap<String, String>,
    /// Scope the artifact to a user.
    pub user_id: Option<String>,
    pub description: Option<String>,
    /// VFS path, e.g. `/projects/alpha/readme.md`.
    pub virtual_path: Option<String>,
}

/// Optional parameters for [`ArtifactClient::list_with`].
#[derive(Debug, Clone, Default)]
pub struct ListOptions {
    /// Filter by source server.
    pub source: Option<String>,
    pub user_id: Option<String>,
    pub limit: Option<i32>,
    pub offset: Option<i32>,
    /// If set, lists the given VFS directory (entries may have `is_directory`).
    pub dir_path: Option<String>,
}

#[derive(Clone)]
pub struct ArtifactClient {
    client: ArtifactServiceClient<Channel>,
    token: Option<String>,
    default_user_id: Option<String>,
    default_source: Option<String>,
}

impl ArtifactClient {
    fn normalize_addr(addr: &str) -> String {
        if addr.contains("://") {
            addr.to_string()
        } else if addr.starts_with(':') {
            format!("http://localhost{addr}")
        } else {
            format!("http://{addr}")
        }
    }

    /// Connect to the artifact service.
    ///
    /// Automatically reads `ARTIFACT_GRPC_TOKEN` / `ARTIFACT_TOKEN`, `ARTIFACT_USER_ID`,
    /// and `ARTIFACT_SOURCE` from the environment if present.
    pub async fn connect(dst: impl Into<String>) -> Result<Self, tonic::transport::Error> {
        let token = std::env::var("ARTIFACT_GRPC_TOKEN")
            .or_else(|_| std::env::var("ARTIFACT_TOKEN"))
            .ok()
            .filter(|s| !s.is_empty());
        Self::connect_with_token(dst, token).await
    }

    /// Connect with an explicit authentication token (or `None`).
    pub async fn connect_with_token(
        dst: impl Into<String>,
        token: Option<String>,
    ) -> Result<Self, tonic::transport::Error> {
        let addr = Self::normalize_addr(&dst.into());
        let client = ArtifactServiceClient::connect(addr).await?;
        let default_user_id = std::env::var("ARTIFACT_USER_ID").ok().filter(|s| !s.is_empty());
        let default_source = std::env::var("ARTIFACT_SOURCE").ok().filter(|s| !s.is_empty());
        Ok(Self {
            client,
            token,
            default_user_id,
            default_source,
        })
    }

    /// Connect using `ARTIFACT_GRPC_ADDR` (defaulting to `http://localhost:9590`).
    pub async fn from_env() -> Result<Self, tonic::transport::Error> {
        let addr = std::env::var("ARTIFACT_GRPC_ADDR").unwrap_or_else(|_| "http://localhost:9590".to_string());
        Self::connect(addr).await
    }

    fn make_request<T>(&self, message: T) -> tonic::Request<T> {
        let mut req = tonic::Request::new(message);
        if let Some(ref token) = self.token {
            if let Ok(val) = format!("Bearer {token}").parse() {
                req.metadata_mut().insert("authorization", val);
            }
        }
        req
    }

    /// Low-level write taking the raw request.
    pub async fn write(&self, request: gen::WriteRequest) -> Result<gen::WriteResponse, tonic::Status> {
        let mut client = self.client.clone();
        let response = client.write(self.make_request(request)).await?;
        Ok(response.into_inner())
    }

    /// Write an artifact; set `opts.virtual_path` to place it in the VFS.
    pub async fn write_file(
        &self,
        filename: impl Into<String>,
        content: impl Into<Vec<u8>>,
        opts: WriteOptions,
    ) -> Result<gen::WriteResponse, tonic::Status> {
        self.write(gen::WriteRequest {
            filename: filename.into(),
            content: content.into(),
            mime_type: opts.mime_type.unwrap_or_default(),
            expires_hours: opts.expires_hours.unwrap_or_default(),
            source: opts.source.or_else(|| self.default_source.clone()).unwrap_or_default(),
            metadata: opts.metadata,
            user_id: opts.user_id.or_else(|| self.default_user_id.clone()).unwrap_or_default(),
            description: opts.description.unwrap_or_default(),
            virtual_path: opts.virtual_path.unwrap_or_default(),
        })
        .await
    }

    /// Read by artifact ID, filename, or virtual path (starting with `/`).
    pub async fn read(&self, id: String, user_id: Option<String>) -> Result<gen::ReadResponse, tonic::Status> {
        let mut client = self.client.clone();
        let request = gen::ReadRequest {
            id,
            user_id: user_id.or_else(|| self.default_user_id.clone()).unwrap_or_default(),
        };
        let response = client.read(self.make_request(request)).await?;
        Ok(response.into_inner())
    }

    /// Flat listing (unchanged signature). Use [`Self::list_with`] for `source`/`dir_path`.
    pub async fn list(&self, user_id: Option<String>, limit: Option<i32>, offset: Option<i32>) -> Result<gen::ListResponse, tonic::Status> {
        self.list_with(ListOptions {
            user_id,
            limit,
            offset,
            ..Default::default()
        })
        .await
    }

    /// Listing with all filters, including VFS directory mode via `dir_path`.
    pub async fn list_with(&self, opts: ListOptions) -> Result<gen::ListResponse, tonic::Status> {
        let mut client = self.client.clone();
        let request = gen::ListRequest {
            source: opts.source.or_else(|| self.default_source.clone()).unwrap_or_default(),
            user_id: opts.user_id.or_else(|| self.default_user_id.clone()).unwrap_or_default(),
            limit: opts.limit.unwrap_or_default(),
            offset: opts.offset.unwrap_or_default(),
            dir_path: opts.dir_path.unwrap_or_default(),
        };
        let response = client.list(self.make_request(request)).await?;
        Ok(response.into_inner())
    }

    /// Delete by artifact ID, filename, or virtual path.
    pub async fn delete(&self, id: String, user_id: Option<String>) -> Result<gen::DeleteResponse, tonic::Status> {
        let mut client = self.client.clone();
        let request = gen::DeleteRequest {
            id,
            user_id: user_id.or_else(|| self.default_user_id.clone()).unwrap_or_default(),
        };
        let response = client.delete(self.make_request(request)).await?;
        Ok(response.into_inner())
    }

    /// Patch an artifact's content.
    ///
    /// With `append = true` the content is appended and the line range is ignored.
    /// Otherwise lines `[line_start, line_end)` (0-based) are replaced by `content`;
    /// `None` is sent as 0.
    pub async fn patch(
        &self,
        id_or_path: impl Into<String>,
        content: impl Into<Vec<u8>>,
        line_start: Option<i32>,
        line_end: Option<i32>,
        append: bool,
        user_id: Option<String>,
    ) -> Result<gen::PatchResponse, tonic::Status> {
        let mut client = self.client.clone();
        let request = gen::PatchRequest {
            id: id_or_path.into(),
            user_id: user_id.or_else(|| self.default_user_id.clone()).unwrap_or_default(),
            content: content.into(),
            line_start: line_start.unwrap_or_default(),
            line_end: line_end.unwrap_or_default(),
            append,
        };
        let response = client.patch(self.make_request(request)).await?;
        Ok(response.into_inner())
    }

    /// Find artifacts whose virtual path matches a glob pattern (e.g. `/logs/*.txt`).
    pub async fn find(&self, pattern: impl Into<String>, user_id: Option<String>) -> Result<gen::ListResponse, tonic::Status> {
        let mut client = self.client.clone();
        let request = gen::FindRequest {
            user_id: user_id.or_else(|| self.default_user_id.clone()).unwrap_or_default(),
            pattern: pattern.into(),
        };
        let response = client.find(self.make_request(request)).await?;
        Ok(response.into_inner())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_normalize_addr() {
        assert_eq!(ArtifactClient::normalize_addr("localhost:9590"), "http://localhost:9590");
        assert_eq!(ArtifactClient::normalize_addr(":9590"), "http://localhost:9590");
        assert_eq!(ArtifactClient::normalize_addr("http://127.0.0.1:9590"), "http://127.0.0.1:9590");
        assert_eq!(ArtifactClient::normalize_addr("https://remote.server:9590"), "https://remote.server:9590");
    }

    #[tokio::test]
    async fn test_make_request_with_token() {
        let dummy_channel = tonic::transport::Endpoint::from_static("http://127.0.0.1:9590").connect_lazy();
        let client = ArtifactClient {
            client: ArtifactServiceClient::new(dummy_channel),
            token: Some("secret123".into()),
            default_user_id: None,
            default_source: None,
        };

        let req = client.make_request("dummy");
        let auth = req.metadata().get("authorization").expect("auth header present");
        assert_eq!(auth.to_str().unwrap(), "Bearer secret123");
    }

    #[tokio::test]
    async fn test_make_request_without_token() {
        let dummy_channel = tonic::transport::Endpoint::from_static("http://127.0.0.1:9590").connect_lazy();
        let client = ArtifactClient {
            client: ArtifactServiceClient::new(dummy_channel),
            token: None,
            default_user_id: None,
            default_source: None,
        };

        let req = client.make_request("dummy");
        assert!(req.metadata().get("authorization").is_none());
    }
}

