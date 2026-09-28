#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

if [ ! -f "$ROOT_DIR/VERSION" ]; then
    echo "Error: $ROOT_DIR/VERSION file not found" >&2
    exit 1
fi

VERSION="$(tr -d '[:space:]' < "$ROOT_DIR/VERSION")"
if [ -z "$VERSION" ]; then
    echo "Error: $ROOT_DIR/VERSION is empty" >&2
    exit 1
fi

echo "Syncing VERSION ($VERSION) across server and client libraries..."

# 1. Go Client (client/client.go)
if [ -f "$ROOT_DIR/client/client.go" ]; then
    sed -i -E "s/const Version = \"[^\"]+\"/const Version = \"$VERSION\"/" "$ROOT_DIR/client/client.go"
    echo "  ✓ client/client.go -> $VERSION"
fi

# 2. Rust Client (client-rust/Cargo.toml & Cargo.lock)
if [ -f "$ROOT_DIR/client-rust/Cargo.toml" ]; then
    sed -i -E "0,/^version = \"[^\"]+\"/s//version = \"$VERSION\"/" "$ROOT_DIR/client-rust/Cargo.toml"
    (cd "$ROOT_DIR/client-rust" && cargo check --quiet 2>/dev/null || cargo check)
    echo "  ✓ client-rust/Cargo.toml & Cargo.lock -> $VERSION"
fi

# 3. Python Client (client-python/pyproject.toml)
if [ -f "$ROOT_DIR/client-python/pyproject.toml" ]; then
    sed -i -E "s/^version = \"[^\"]+\"/version = \"$VERSION\"/" "$ROOT_DIR/client-python/pyproject.toml"
    echo "  ✓ client-python/pyproject.toml -> $VERSION"
fi

# 4. TypeScript Client (client-ts/package.json)
if [ -f "$ROOT_DIR/client-ts/package.json" ]; then
    (cd "$ROOT_DIR/client-ts" && npm version "$VERSION" --no-git-tag-version --allow-same-version --silent)
    echo "  ✓ client-ts/package.json -> $VERSION"
fi

# 5. Go CLI & Server default fallback (cmd/artifact-cli/main.go & cmd/artifact-server/main.go)
if [ -f "$ROOT_DIR/cmd/artifact-cli/main.go" ]; then
    sed -i -E "s/var version = \"[^\"]+\"/var version = \"$VERSION\"/" "$ROOT_DIR/cmd/artifact-cli/main.go"
    echo "  ✓ cmd/artifact-cli/main.go -> $VERSION"
fi
if [ -f "$ROOT_DIR/cmd/artifact-server/main.go" ]; then
    sed -i -E "s/version = \"[^\"]+\"/version = \"$VERSION\"/" "$ROOT_DIR/cmd/artifact-server/main.go"
    echo "  ✓ cmd/artifact-server/main.go -> $VERSION"
fi

echo "Version sync completed successfully: $VERSION"
