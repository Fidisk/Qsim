#!/usr/bin/env bash
# Build the project to WebAssembly using the local Raylib-Go-Wasm fork.
#
# Produces:
#   Raylib-Go-Wasm/index/main.wasm    - the app
#   Raylib-Go-Wasm/index/wasm_exec.js - Go wasm runtime glue (refreshed each run)
#   server.exe (project root, first run only) - dev server for the index folder
#
# go.mod is temporarily pointed at the fork and restored when the script
# exits, so desktop builds keep working. First run needs network access to
# download github.com/BrownNPC/wasm-ffi-go.
#
# Usage:  script/build-wasm.sh        (Git Bash)
# Then:   ./server.exe                (from the project root)
#         open http://localhost:8080
set -euo pipefail

cd "$(dirname "$0")/.."

if [ ! -d "Raylib-Go-Wasm/raylib" ]; then
	echo "error: Raylib-Go-Wasm fork not found in $(pwd)" >&2
	exit 1
fi

cp go.mod go.mod.wasm.bak
cp go.sum go.sum.wasm.bak
restore() {
	mv go.mod.wasm.bak go.mod
	mv go.sum.wasm.bak go.sum
}
trap restore EXIT

echo "==> go mod: replacing raylib with the Raylib-Go-Wasm fork"
go mod edit \
	-replace github.com/gen2brain/raylib-go/raylib=./Raylib-Go-Wasm/raylib \
	-replace github.com/BrownNPC/Raylib-Go-Wasm/wasm-runtime=./Raylib-Go-Wasm/wasm-runtime
go mod tidy

echo "==> building wasm"
GOOS=js GOARCH=wasm go build -o ./Raylib-Go-Wasm/index/main.wasm .

echo "==> copying wasm_exec.js from GOROOT"
cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" ./Raylib-Go-Wasm/index/wasm_exec.js

if [ ! -f server.exe ]; then
	echo "==> building dev server (first time only)"
	go build -o server.exe ./Raylib-Go-Wasm/server/server.go
fi

echo
echo "Done. Start the app with:  ./server.exe"
echo "Then open:                 http://localhost:8080"
