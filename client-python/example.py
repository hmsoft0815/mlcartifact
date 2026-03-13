## This file is part of the mlcartifact server examples.
from mlcartifact import ArtifactClient
import os

def main():
    # Use environment variables or pass address directly
    addr = os.getenv("ARTIFACT_GRPC_ADDR") or "localhost:9590"
    
    with ArtifactClient(addr) as client:
        print(f"Connecting to {client.addr}...")
        print("--- mlcartifact Python 'Hello World' Example ---")

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
        except Exception as e:
            if "not found" in str(e).lower() or "not_found" in str(e).lower():
                print("Verified: Artifact 2 is indeed gone.")
            else:
                raise e

        print("--- Example finished successfully ---")

if __name__ == "__main__":
    main()
