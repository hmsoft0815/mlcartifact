## This file is part of the mlcartifact server examples.
import os
import uuid

from mlcartifact import ArtifactClient, ArtifactError


def main():
    # Use environment variables or pass address directly
    addr = os.getenv("ARTIFACT_GRPC_ADDR") or "localhost:9590"

    with ArtifactClient(addr) as client:
        print(f"Connecting to {client.addr}...")
        print("--- mlcartifact Python example ---")

        # 1. Write 3 artifacts
        items = [
            ("artifact1.txt", b"Content for artifact A"),
            ("artifact2.txt", b"Content for artifact B"),
            ("artifact3.txt", b"Content for artifact C"),
        ]

        ids = []
        for name, content in items:
            res = client.write(filename=name, content=content, source="python-example")
            ids.append(res.id)
            print(f"Wrote: {name} (ID: {res.id})")

        # 2. Delete one (artifact 2)
        print(f"Deleting artifact 2 (ID: {ids[1]})...")
        client.delete(ids[1])

        # 3. Retrieve others and compare
        for i in [0, 2]:
            read_res = client.read(ids[i])
            if read_res.content != items[i][1]:
                raise Exception(f"Content mismatch for {items[i][0]}! Expected '{items[i][1].decode('utf-8')}', got '{read_res.content.decode('utf-8')}'")
            print(f"Verified: {items[i][0]} (ID: {ids[i]}) content matches.")

        # 4. Verify artifact 2 is gone
        try:
            client.read(ids[1])
            raise Exception("Error: Artifact 2 should have been deleted but was found!")
        except ArtifactError as e:
            if e.code != "not_found":
                raise
            print("Verified: Artifact 2 is indeed gone.")

        # 5. Virtual file system: write under a path, list a directory, patch, find
        root = f"/py-example-{uuid.uuid4().hex[:8]}"
        readme = f"{root}/docs/readme.md"
        client.write(filename="readme.md", content=b"line 1\nline 2\nline 3",
                     virtual_path=readme, metadata={"lang": "en"}, source="python-example")
        client.write(filename="notes.txt", content=b"notes", virtual_path=f"{root}/notes.txt",
                     source="python-example")
        print(f"Wrote VFS files under {root}")

        listing = client.list(dir_path=root)
        entries = sorted((it.virtual_path or it.filename, it.is_directory) for it in listing.items)
        print(f"list(dir_path={root!r}): {entries}")
        if not any(is_dir for _, is_dir in entries) or len(entries) != 2:
            raise Exception("Expected one file and one sub-directory in the VFS listing")

        client.patch(readme, b"LINE 2", line_start=1, line_end=2)   # replace line 2
        client.patch(readme, b"\nline 4", append=True)               # append
        patched = client.read(readme).content
        if patched != b"line 1\nLINE 2\nline 3\nline 4":
            raise Exception(f"Unexpected content after patch: {patched!r}")
        print(f"Patched {readme}: {patched!r}")

        found = client.find(f"{root}/*/*.md")
        paths = [it.virtual_path for it in found.items]
        if readme not in paths:
            raise Exception(f"find() did not return {readme}: {paths}")
        print(f"find('{root}/*/*.md'): {paths}")

        # Cleanup
        for path in (readme, f"{root}/notes.txt"):
            client.delete(path)
        for i in (0, 2):
            client.delete(ids[i])

        print("--- Example finished successfully ---")


if __name__ == "__main__":
    main()
