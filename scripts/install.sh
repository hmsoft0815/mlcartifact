#!/bin/sh
set -e

# mlcartifact Ecosystem Installer
# Usage: curl -sfL https://raw.githubusercontent.com/hmsoft0815/mlcartifact/main/scripts/install.sh | sh

GITHUB_REPO="hmsoft0815/mlcartifact"
INSTALL_DIR="/usr/local/bin"

# Detect OS
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$OS" in
  linux*)  OS='linux';;
  darwin*) OS='darwin';;
  msys*|mingw*) OS='windows';;
  *) echo "Unsupported OS: $OS"; exit 1;;
esac

# Detect Arch
ARCH=$(uname -m)
case "$ARCH" in
  x86_64) ARCH='amd64';;
  arm64|aarch64) ARCH='arm64';;
  *) echo "Unsupported Architecture: $ARCH"; exit 1;;
esac

echo "🚀 Installing mlcartifact for $OS/$ARCH..."

# Get latest release tag
TAG=$(curl -sfL "https://api.github.com/repos/$GITHUB_REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$TAG" ]; then
  echo "❌ Could not find latest release tag."
  exit 1
fi

echo "📦 Found version $TAG"

# Download URL template
# Example: mlcartifact_0.1.0_linux_amd64.tar.gz
BINARY_URL="https://github.com/$GITHUB_REPO/releases/download/$TAG/mlcartifact_${TAG#v}_${OS}_${ARCH}.tar.gz"
if [ "$OS" = "windows" ]; then
    BINARY_URL="https://github.com/$GITHUB_REPO/releases/download/$TAG/mlcartifact_${TAG#v}_${OS}_${ARCH}.zip"
fi

TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT

echo "⬇️ Downloading from $BINARY_URL..."
curl -sfL "$BINARY_URL" -o "$TMP_DIR/bundle.archive"

echo "📂 Extracting binaries..."
if [ "$OS" = "windows" ]; then
    unzip -q "$TMP_DIR/bundle.archive" -d "$TMP_DIR"
else
    tar -xzf "$TMP_DIR/bundle.archive" -C "$TMP_DIR"
fi

# Move binaries to install dir
BINARIES="artifact-server artifact-cli"
for bin in $BINARIES; do
    if [ "$OS" = "windows" ]; then
        mv "$TMP_DIR/$bin.exe" "$INSTALL_DIR/" || true
    else
        echo "Installing $bin to $INSTALL_DIR/$bin..."
        sudo mv "$TMP_DIR/$bin" "$INSTALL_DIR/"
        sudo chmod +x "$INSTALL_DIR/$bin"
    fi
done

echo "✅ Installation complete! You can now use: $BINARIES"
