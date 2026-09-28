import os
import unittest
import httpx

from mlcartifact import ArtifactClient, ArtifactError
from mlcartifact.gen import artifact_pb2 as pb


class TestArtifactClient(unittest.TestCase):
    def test_init_defaults(self):
        client = ArtifactClient("localhost:9590")
        self.assertEqual(client.addr, "http://localhost:9590")
        self.assertEqual(client.base_url, "http://localhost:9590")
        self.assertEqual(client.token, "")
        client.close()

    def test_init_with_http_prefix(self):
        client = ArtifactClient("https://remote.server:9590/")
        self.assertEqual(client.addr, "https://remote.server:9590/")
        self.assertEqual(client.base_url, "https://remote.server:9590")
        client.close()

    def test_init_env_vars(self):
        os.environ["ARTIFACT_GRPC_ADDR"] = "custom-host:9590"
        os.environ["ARTIFACT_GRPC_TOKEN"] = "test-token"
        os.environ["ARTIFACT_SOURCE"] = "test-source"
        os.environ["ARTIFACT_USER_ID"] = "user-123"
        try:
            client = ArtifactClient()
            self.assertEqual(client.addr, "http://custom-host:9590")
            self.assertEqual(client.token, "test-token")
            self.assertEqual(client.default_source, "test-source")
            self.assertEqual(client.default_user_id, "user-123")
            client.close()
        finally:
            del os.environ["ARTIFACT_GRPC_ADDR"]
            del os.environ["ARTIFACT_GRPC_TOKEN"]
            del os.environ["ARTIFACT_SOURCE"]
            del os.environ["ARTIFACT_USER_ID"]

    def test_auth_header_and_mock_call(self):
        captured_headers = {}

        def handler(request: httpx.Request) -> httpx.Response:
            nonlocal captured_headers
            captured_headers = dict(request.headers)

            if "Write" in str(request.url):
                req = pb.WriteRequest()
                req.ParseFromString(request.content)
                self.assertEqual(req.filename, "test.txt")
                self.assertEqual(req.content, b"data")

                resp = pb.WriteResponse(
                    id="mock-id-123",
                    filename="test.txt",
                    uri="mlcartifact://mock-id-123",
                )
                return httpx.Response(200, content=resp.SerializeToString())

            return httpx.Response(404, text="Not Found")

        transport = httpx.MockTransport(handler)
        mock_http = httpx.Client(transport=transport)

        with ArtifactClient("localhost:9590", token="secret-token", client=mock_http) as client:
            res = client.write(filename="test.txt", content=b"data")
            self.assertEqual(res.id, "mock-id-123")
            self.assertEqual(res.uri, "mlcartifact://mock-id-123")
            self.assertEqual(captured_headers.get("authorization"), "Bearer secret-token")
            self.assertEqual(captured_headers.get("connect-protocol-version"), "1")
            self.assertEqual(captured_headers.get("content-type"), "application/proto")

    def test_error_handling(self):
        def handler(request: httpx.Request) -> httpx.Response:
            return httpx.Response(
                404,
                headers={"Content-Type": "application/json"},
                json={"code": "not_found", "message": "artifact not found"}
            )

        transport = httpx.MockTransport(handler)
        mock_http = httpx.Client(transport=transport)

        with ArtifactClient("localhost:9590", client=mock_http) as client:
            with self.assertRaises(ArtifactError) as ctx:
                client.read("missing-id")
            self.assertEqual(ctx.exception.code, "not_found")
            self.assertIn("artifact not found", ctx.exception.message)


if __name__ == "__main__":
    unittest.main()
