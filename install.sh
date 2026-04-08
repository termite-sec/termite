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
  linux) BINARY="termite-linux-$ARCH" ;;
  darwin) BINARY="termite-darwin-$ARCH" ;;
  *) echo "Unsupported OS: $OS"; exit 1 ;;
esac

echo "Downloading termite for $OS/$ARCH..."
curl -sSL "https://github.com/termite-sec/termite/releases/latest/download/$BINARY" -o /tmp/termite
chmod +x /tmp/termite
sudo mv /tmp/termite /usr/local/bin/termite

echo "termite installed successfully!"
termite --version
