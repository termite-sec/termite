#!/bin/bash

set -e

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case $ARCH in
  x86_64) ARCH="amd64" ;;
  arm64|aarch64) ARCH="arm64" ;;
  *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

case $OS in
  linux) BINARY="kitin-linux-$ARCH" ;;
  darwin) BINARY="kitin-darwin-$ARCH" ;;
  *) echo "Unsupported OS: $OS"; exit 1 ;;
esac

echo "Downloading kitin for $OS/$ARCH..."
curl -sSL "https://github.com/kitin-sec/kitin/releases/latest/download/$BINARY" -o /tmp/kitin
chmod +x /tmp/kitin
sudo mv /tmp/kitin /usr/local/bin/kitin

echo "kitin installed successfully!"
kitin --version
