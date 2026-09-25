//! Rust client for the mlcartifact service (gRPC, `artifact.v1`).
//!
//! The raw generated types and client live in [`gen`]; [`ArtifactClient`] is a
//! thin convenience wrapper around them.

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

pub struct ArtifactClient {
    client: ArtifactServiceClient<Channel>,
}

impl ArtifactClient {
    pub async fn connect(dst: String) -> Result<Self, tonic::transport::Error> {
        let client = ArtifactServiceClient::connect(dst).await?;
        Ok(Self { client })
    }

    /// Low-level write taking the raw request.
    pub async fn write(&self, request: gen::WriteRequest) -> Result<gen::WriteResponse, tonic::Status> {
        let mut client = self.client.clone();
        let response = client.write(request).await?;
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
            source: opts.source.unwrap_or_default(),
            metadata: opts.metadata,
            user_id: opts.user_id.unwrap_or_default(),
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
            user_id: user_id.unwrap_or_default(),
        };
        let response = client.read(request).await?;
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
            source: opts.source.unwrap_or_default(),
            user_id: opts.user_id.unwrap_or_default(),
            limit: opts.limit.unwrap_or_default(),
            offset: opts.offset.unwrap_or_default(),
            dir_path: opts.dir_path.unwrap_or_default(),
        };
        let response = client.list(request).await?;
        Ok(response.into_inner())
    }

    /// Delete by artifact ID, filename, or virtual path.
    pub async fn delete(&self, id: String, user_id: Option<String>) -> Result<gen::DeleteResponse, tonic::Status> {
        let mut client = self.client.clone();
        let request = gen::DeleteRequest {
            id,
            user_id: user_id.unwrap_or_default(),
        };
        let response = client.delete(request).await?;
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
            user_id: user_id.unwrap_or_default(),
            content: content.into(),
            line_start: line_start.unwrap_or_default(),
            line_end: line_end.unwrap_or_default(),
            append,
        };
        let response = client.patch(request).await?;
        Ok(response.into_inner())
    }

    /// Find artifacts whose virtual path matches a glob pattern (e.g. `/logs/*.txt`).
    pub async fn find(&self, pattern: impl Into<String>, user_id: Option<String>) -> Result<gen::ListResponse, tonic::Status> {
        let mut client = self.client.clone();
        let request = gen::FindRequest {
            user_id: user_id.unwrap_or_default(),
            pattern: pattern.into(),
        };
        let response = client.find(request).await?;
        Ok(response.into_inner())
    }
}
