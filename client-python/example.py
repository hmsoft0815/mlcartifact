## This file is part of the mlcartifact server examples.
from mlcartifact import ArtifactClient
import os

def main():
    # Use environment variables or pass address directly
    addr = os.getenv("ARTIFACT_GRPC_ADDR") or "localhost:9590"
    
    with ArtifactClient(addr) as client:
        print(f"Connecting to {client.addr}...")
        
        # 1. Write an artifact
        filename = "python_test.md"
        content = b"# Hello from Python\nThis is an artifact saved via the Python Connect client."
        
        try:
            res = client.write(
                filename=filename,
                content=content,
                description="Saved from Python example",
                source="python-client"
            )
            print("Successfully saved artifact:")
            print(f"  ID: {res.id}")
            print(f"  URI: {res.uri}")
            
            # 2. Read it back
            print(f"\nReading artifact {res.id}...")
            read_res = client.read(res.id)
            print(f"  Filename: {read_res.filename}")
            print(f"  Content: {read_res.content.decode('utf-8')}")
            
            # 3. List artifacts
            print("\nListing artifacts...")
            list_res = client.list(limit=5)
            for item in list_res.items:
                print(f"  - {item.id}: {item.filename} ({item.size_bytes} bytes)")
                
        except Exception as e:
            print(f"Error: {e}")

if __name__ == "__main__":
    main()
