import { Code, ConnectError } from "@connectrpc/connect";
import { ArtifactClient } from "../src/client.js";

function check(cond: unknown, msg: string): asserts cond {
    if (!cond) throw new Error(`Check failed: ${msg}`);
}

async function main() {
    const addr = process.env.ARTIFACT_GRPC_ADDR || "http://localhost:9590";
    console.log(`Connecting to ${addr}...`);
    console.log("--- mlcartifact TypeScript example ---");

    const client = new ArtifactClient(addr);
    const run = Date.now().toString(36);
    const dir = `/ts-example-${run}`;

    // 1. Write 3 artifacts, each with a virtual path
    const items = [
        { name: "artifact1.txt", content: "Content for artifact A" },
        { name: "artifact2.txt", content: "Content for artifact B" },
        { name: "notes.md", content: "line 0\nline 1\nline 2" },
    ];
    const ids: string[] = [];
    for (const item of items) {
        const res = await client.write(item.name, item.content, {
            source: "ts-example",
            virtualPath: `${dir}/${item.name === "notes.md" ? "docs/" : ""}${item.name}`,
        });
        ids.push(res.id);
        console.log(`Wrote: ${item.name} (ID: ${res.id}, path: ${res.virtualPath})`);
    }

    // 2. Read back by ID and by virtual path
    for (const [idx, key] of [[0, ids[0]], [1, `${dir}/artifact2.txt`]] as const) {
        const text = new TextDecoder().decode((await client.read(key)).content);
        check(text === items[idx].content, `content of ${key}`);
        console.log(`Verified read: ${key}`);
    }

    // 3. VFS directory listing
    const listing = await client.list({ dirPath: dir });
    for (const it of listing.items) {
        console.log(`  ${it.isDirectory ? "[dir] " : "      "}${it.virtualPath || it.filename}`);
    }
    check(listing.items.some((i) => i.isDirectory), "list(dirPath) returns the docs/ sub-directory");
    check(listing.items.some((i) => i.filename === "artifact1.txt"), "list(dirPath) returns artifact1.txt");

    // 4. Patch: append, then replace line 1
    const notes = `${dir}/docs/notes.md`;
    await client.patch(notes, "\nline 3", { append: true });
    const p = await client.patch(notes, "LINE ONE", { lineStart: 1, lineEnd: 2 });
    const patched = new TextDecoder().decode((await client.read(notes)).content);
    check(patched === "line 0\nLINE ONE\nline 2\nline 3", `patched content, got ${JSON.stringify(patched)}`);
    console.log(`Verified patch (success=${p.success}, newSize=${p.newSize})`);

    // 5. Find by glob / substring
    const found = await client.find(`${dir}/*.txt`);
    console.log(`Find ${dir}/*.txt -> ${found.items.map((i) => i.virtualPath).join(", ")}`);
    check(found.items.length === 2, "find returns the two .txt artifacts");

    // 6. Delete (by ID and by path) and verify it is gone
    await client.delete(ids[1]);
    try {
        await client.read(ids[1]);
        throw new Error("Artifact 2 should have been deleted but was found!");
    } catch (e) {
        const err = ConnectError.from(e);
        if (err.code !== Code.NotFound && !err.message.toLowerCase().includes("not found")) throw e;
        console.log("Verified: artifact 2 is gone.");
    }
    await client.delete(ids[0]);
    await client.delete(notes);

    console.log("--- Example finished successfully ---");
}

main().catch((err) => {
    console.error("Example failed:", err);
    process.exit(1);
});
