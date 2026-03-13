import { ArtifactClient } from "../src/client.js";

async function main() {
    const addr = process.env.ARTIFACT_GRPC_ADDR || "http://localhost:9590";
    console.log(`Connecting to ${addr}...`);
    console.log("--- mlcartifact TypeScript 'Hello World' Example ---");

    const client = new ArtifactClient(addr);

    // 1. Write 3 artifacts
    const items = [
        { name: "artifact1.txt", content: "Content for artifact A" },
        { name: "artifact2.txt", content: "Content for artifact B" },
        { name: "artifact3.txt", content: "Content for artifact C" },
    ];

    const ids: string[] = [];
    for (const item of items) {
        const res = await client.write(item.name, item.content, { source: "ts-example" });
        ids.push(res.id);
        console.log(`Wrote: ${item.name} (ID: ${res.id})`);
    }

    // 2. Delete one (artifact 2)
    console.log(`Deleting artifact 2 (ID: ${ids[1]})...`);
    await client.delete(ids[1]);

    // 3. Retrieve others and compare
    const toCheck = [0, 2];
    for (const idx of toCheck) {
        const readRes = await client.read(ids[idx]);
        const textContent = new TextDecoder().decode(readRes.content);
        
        if (textContent !== items[idx].content) {
            throw new Error(`Content mismatch for ${items[idx].name}! Expected '${items[idx].content}', got '${textContent}'`);
        }
        console.log(`Verified: ${items[idx].name} (ID: ${ids[idx]}) content matches.`);
    }

    // 4. Verify artifact 2 is gone
    try {
        await client.read(ids[1]);
        throw new Error("Error: Artifact 2 should have been deleted but was found!");
    } catch (e: any) {
        if (e.message?.toLowerCase().includes("not found") || e.code === 5 /* NotFound */) {
            console.log("Verified: Artifact 2 is indeed gone.");
        } else {
            throw e;
        }
    }

    console.log("--- Example finished successfully ---");
}

main().catch((err) => {
    console.error("Example failed:", err);
    process.exit(1);
});
