import { describe, it, expect } from "vitest";
import { create } from "@bufbuild/protobuf";
import { createRouterTransport } from "@connectrpc/connect";
import { ArtifactClient } from "./client.js";
import {
  ArtifactService,
  ArtifactInfoSchema,
  type WriteRequest,
  type ListRequest,
  type PatchRequest,
} from "./gen/artifact_pb.js";

describe("ArtifactClient", () => {
  it("writes an artifact (incl. virtualPath)", async () => {
    let seen: WriteRequest | undefined;
    const transport = createRouterTransport(({ service }) => {
      service(ArtifactService, {
        write(req) {
          seen = req;
          return {
            id: "mock-id-123",
            filename: req.filename,
            uri: `artifact://${req.filename}`,
            expiresAt: new Date().toISOString(),
            virtualPath: req.virtualPath,
          };
        },
      });
    });

    const client = new ArtifactClient("http://localhost:9590", transport);
    const resp = await client.write("test.txt", "Hello Mock!", { virtualPath: "/docs/test.txt" });

    expect(resp.id).toBe("mock-id-123");
    expect(resp.uri).toBe("artifact://test.txt");
    expect(resp.virtualPath).toBe("/docs/test.txt");
    expect(new TextDecoder().decode(seen!.content)).toBe("Hello Mock!");
  });

  it("accepts Blob content", async () => {
    let bytes = "";
    const transport = createRouterTransport(({ service }) => {
      service(ArtifactService, {
        write(req) {
          bytes = new TextDecoder().decode(req.content);
          return { id: "b" };
        },
      });
    });
    const client = new ArtifactClient(undefined, transport);
    await client.write("b.txt", new Blob(["from blob"]));
    expect(bytes).toBe("from blob");
  });

  it("reads an artifact", async () => {
    const transport = createRouterTransport(({ service }) => {
      service(ArtifactService, {
        read() {
          return {
            content: new TextEncoder().encode("Mocked content"),
            mimeType: "text/plain",
            filename: "mock.txt",
          };
        },
      });
    });

    const client = new ArtifactClient("http://localhost:9590", transport);
    const resp = await client.read("some-id");

    expect(new TextDecoder().decode(resp.content)).toBe("Mocked content");
    expect(resp.mimeType).toBe("text/plain");
  });

  it("passes dirPath to list", async () => {
    let seen: ListRequest | undefined;
    const transport = createRouterTransport(({ service }) => {
      service(ArtifactService, {
        list(req) {
          seen = req;
          return { items: [create(ArtifactInfoSchema, { filename: "sub", isDirectory: true })] };
        },
      });
    });
    const client = new ArtifactClient(undefined, transport);
    const resp = await client.list({ dirPath: "/projects" });
    expect(seen!.dirPath).toBe("/projects");
    expect(resp.items[0].isDirectory).toBe(true);
  });

  it("patches and finds", async () => {
    let patchReq: PatchRequest | undefined;
    let pattern = "";
    const transport = createRouterTransport(({ service }) => {
      service(ArtifactService, {
        patch(req) {
          patchReq = req;
          return { success: true, newSize: 42n };
        },
        find(req) {
          pattern = req.pattern;
          return { items: [{ virtualPath: "/a/b.md" }] };
        },
      });
    });
    const client = new ArtifactClient(undefined, transport);

    const p = await client.patch("/a/b.md", "more", { append: true });
    expect(p.success).toBe(true);
    expect(p.newSize).toBe(42n);
    expect(patchReq!.append).toBe(true);
    expect(patchReq!.id).toBe("/a/b.md");

    const f = await client.find("*.md");
    expect(pattern).toBe("*.md");
    expect(f.items[0].virtualPath).toBe("/a/b.md");
  });
});
