#!/bin/bash

set -e

echo "Building Shelly..."
go build -o shelly main.go

echo "Making binary executable..."
chmod +x shelly

echo "Installing to /usr/local/bin/ (requires sudo)..."
sudo mv shelly /usr/local/bin/

echo "Done. To get started, run \`shelly --help\`"

