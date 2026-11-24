#!/usr/bin/env bash
set -euo pipefail

if [ $# -lt 1 ]; then
  echo "Usage: $0 <browser>"
  echo ""
  echo "Installs cfutils and registers it as an MCP server in OpenCode."
  echo ""
  echo "Arguments:"
  echo "  browser   Browser name (chrome, brave, edge, etc.)"
  exit 1
fi

BROWSER="$1"
BINDIR="$HOME/.local/bin"
mkdir -p "$BINDIR"

echo "Building cfutils..."
go build -o "$BINDIR/cfutils" ./cmd

echo "Installed: $BINDIR/cfutils"
echo ""
echo "Registering codefresh in OpenCode..."

opencode mcp add --global codefresh -- "$BINDIR/cfutils" mcp --browser "$BROWSER"

echo ""
echo "Restarting OpenCode to apply changes..."
opencode service restart

echo ""
echo "Done! codefresh is now available as an MCP server in OpenCode."