import os
from typing import Any, Dict, Optional

import httpx

from .gen import artifact_pb2 as pb

_SERVICE_PATH = "artifact.v1.ArtifactService"


class ArtifactError(httpx.HTTPStatusError):
    """ Raised when the server answers with a Connect error.

    Subclasses httpx.HTTPStatusError, so existing ``except httpx.HTTPStatusError``
    handlers keep working. ``code`` is the Connect error code (e.g. "not_found"),
    ``message`` the server's error message.
    """

    def __init__(self, code: str, message: str, *, request: httpx.Request, response: httpx.Response):
        self.code = code
        self.message = message
        super().__init__(f"{code}: {message} (HTTP {response.status_code})", request=request, response=response)


class ArtifactClient:
    """ Python client for the mlcartifact service using the Connect protocol
    (binary protobuf over plain HTTP POST; no grpcio needed). """

    def __init__(self, addr: Optional[str] = None):
        """
        Initialize the client.
        :param addr: The address of the artifact server (e.g. 'localhost:9590').
                     If None, it reads from ARTIFACT_GRPC_ADDR environment variable.
        """
        self.addr = addr or os.getenv("ARTIFACT_GRPC_ADDR") or "localhost:9590"
        if not self.addr.startswith(("http://", "https://")):
            self.addr = f"http://{self.addr}"

        self.base_url = self.addr.rstrip("/")
        # HTTP/2 is negotiated via ALPN on https://; plain http:// uses HTTP/1.1,
        # which the Connect protocol supports as well.
        self.client = httpx.Client(http2=True)
        self.default_source = os.getenv("ARTIFACT_SOURCE", "")
        self.default_user_id = os.getenv("ARTIFACT_USER_ID", "")

    def _uid(self, user_id: Optional[str]) -> str:
        return user_id if user_id is not None else self.default_user_id

    def _call(self, method: str, request_msg: Any, response_msg_type: Any) -> Any:
        """ Internal helper to perform a Connect unary RPC call. """
        url = f"{self.base_url}/{_SERVICE_PATH}/{method}"
        headers = {
            "Content-Type": "application/proto",
            "Connect-Protocol-Version": "1",
        }

        resp = self.client.post(url, content=request_msg.SerializeToString(), headers=headers)
        if resp.status_code != 200:
            # Connect errors are JSON: {"code": "...", "message": "..."}
            code, message = "unknown", resp.text
            try:
                body = resp.json()
                code = body.get("code", code)
                message = body.get("message", message)
            except ValueError:
                pass
            raise ArtifactError(code, message, request=resp.request, response=resp)

        response_msg = response_msg_type()
        response_msg.ParseFromString(resp.content)
        return response_msg

    def write(self,
              filename: str,
              content: bytes,
              description: str = "",
              user_id: Optional[str] = None,
              source: Optional[str] = None,
              expires_in_hours: int = 24,
              mime_type: str = "",
              virtual_path: str = "",
              metadata: Optional[Dict[str, str]] = None) -> pb.WriteResponse:
        """ Saves an artifact to the store.

        :param virtual_path: optional VFS path, e.g. "/projects/alpha/readme.md".
        :param metadata: optional string key/value pairs stored with the artifact.
        """
        req = pb.WriteRequest(
            filename=filename,
            content=content,
            description=description,
            user_id=self._uid(user_id),
            source=source if source is not None else self.default_source,
            expires_hours=int(expires_in_hours),
            mime_type=mime_type,
            virtual_path=virtual_path,
            metadata=metadata or {},
        )
        return self._call("Write", req, pb.WriteResponse)

    def read(self, id_or_filename: str, user_id: Optional[str] = None) -> pb.ReadResponse:
        """ Retrieves an artifact by ID, filename or virtual path (starting with "/"). """
        req = pb.ReadRequest(id=id_or_filename, user_id=self._uid(user_id))
        return self._call("Read", req, pb.ReadResponse)

    def list(self,
             user_id: Optional[str] = None,
             limit: int = 0,
             offset: int = 0,
             source: str = "",
             dir_path: str = "") -> pb.ListResponse:
        """ Lists artifacts.

        :param source: optional filter by source tag.
        :param dir_path: if set, lists the direct children of this VFS directory;
                         sub-folders come back with ``is_directory=True``.
        """
        req = pb.ListRequest(
            user_id=self._uid(user_id),
            limit=limit,
            offset=offset,
            source=source,
            dir_path=dir_path,
        )
        return self._call("List", req, pb.ListResponse)

    def delete(self, id_or_filename: str, user_id: Optional[str] = None) -> pb.DeleteResponse:
        """ Deletes an artifact by ID, filename or virtual path. """
        req = pb.DeleteRequest(id=id_or_filename, user_id=self._uid(user_id))
        return self._call("Delete", req, pb.DeleteResponse)

    def patch(self,
              id_or_path: str,
              content: bytes,
              user_id: Optional[str] = None,
              line_start: int = 0,
              line_end: int = 0,
              append: bool = False) -> pb.PatchResponse:
        """ Modifies an artifact in place.

        With ``append=True`` the content is appended to the end. Otherwise the
        lines ``[line_start, line_end)`` (0-based, end exclusive) are replaced by
        ``content``; ``line_start == line_end`` inserts before that line.
        """
        req = pb.PatchRequest(
            id=id_or_path,
            user_id=self._uid(user_id),
            content=content,
            line_start=line_start,
            line_end=line_end,
            append=append,
        )
        return self._call("Patch", req, pb.PatchResponse)

    def find(self, pattern: str, user_id: Optional[str] = None) -> pb.ListResponse:
        """ Searches artifacts by virtual path glob pattern, e.g. "/projects/*/readme.md". """
        req = pb.FindRequest(pattern=pattern, user_id=self._uid(user_id))
        return self._call("Find", req, pb.ListResponse)

    def close(self):
        """ Closes the underlying HTTP client. """
        self.client.close()

    def __enter__(self):
        return self

    def __exit__(self, exc_type, exc_val, exc_tb):
        self.close()
