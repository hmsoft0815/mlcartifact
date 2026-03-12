import os
import httpx
from typing import Optional, List, Dict, Any
from .gen import artifact_pb2 as pb

class ArtifactClient:
    """ Python client for the mlcartifact service using the Connect protocol. """

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
        self.client = httpx.Client(http2=True)
        self.default_source = os.getenv("ARTIFACT_SOURCE", "")
        self.default_user_id = os.getenv("ARTIFACT_USER_ID", "")

    def _call(self, method: str, request_msg: Any, response_msg_type: Any) -> Any:
        """ Internal helper to perform a Connect RPC call. """
        url = f"{self.base_url}/artifact.v1.ArtifactService/{method}"
        headers = {
            "Content-Type": "application/proto",
            "Connect-Protocol-Version": "1",
        }
        
        # Connect protocol expects a binary message
        data = request_msg.SerializeToString()
        
        resp = self.client.post(url, content=data, headers=headers)
        resp.raise_for_status()
        
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
              mime_type: str = "") -> pb.WriteResponse:
        """ Saves an artifact to the store. """
        req = pb.WriteRequest(
            filename=filename,
            content=content,
            description=description,
            user_id=user_id if user_id is not None else self.default_user_id,
            source=source if source is not None else self.default_source,
            expires_hours=int(expires_in_hours),
            mime_type=mime_type
        )
        return self._call("Write", req, pb.WriteResponse)

    def read(self, id_or_filename: str, user_id: Optional[str] = None) -> pb.ReadResponse:
        """ Retrieves an artifact by ID or filename. """
        req = pb.ReadRequest(
            id=id_or_filename,
            user_id=user_id if user_id is not None else self.default_user_id
        )
        return self._call("Read", req, pb.ReadResponse)

    def list(self, 
             user_id: Optional[str] = None, 
             limit: int = 0, 
             offset: int = 0) -> pb.ListResponse:
        """ Lists artifacts. """
        req = pb.ListRequest(
            user_id=user_id if user_id is not None else self.default_user_id,
            limit=limit,
            offset=offset
        )
        return self._call("List", req, pb.ListResponse)

    def delete(self, id_or_filename: str, user_id: Optional[str] = None) -> pb.DeleteResponse:
        """ Deletes an artifact. """
        req = pb.DeleteRequest(
            id=id_or_filename,
            user_id=user_id if user_id is not None else self.default_user_id
        )
        return self._call("Delete", req, pb.DeleteResponse)

    def close(self):
        """ Closes the underlying HTTP client. """
        self.client.close()

    def __enter__(self):
        return self

    def __exit__(self, exc_type, exc_val, exc_tb):
        self.close()
