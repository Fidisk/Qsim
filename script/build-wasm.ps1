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
# Usage:  powershell -File script/build-wasm.ps1
# Then:   ./server.exe                (from the project root)
#         open http://localhost:8080
$ErrorActionPreference = 'Stop'

Set-Location (Split-Path $PSScriptRoot -Parent)

# NOTE: go flags are quoted so PowerShell treats them as values, not as
# parameters of Invoke-Go.
function Invoke-Go {
	param([Parameter(ValueFromRemainingArguments = $true)][string[]]$GoArgs)
	& go @GoArgs
	if ($LASTEXITCODE -ne 0) { throw "go $($GoArgs -join ' ') failed with exit code $LASTEXITCODE" }
}

if (-not (Test-Path 'Raylib-Go-Wasm/raylib')) {
	Write-Error "Raylib-Go-Wasm fork not found in $(Get-Location)"
	exit 1
}

Copy-Item go.mod go.mod.wasm.bak
Copy-Item go.sum go.sum.wasm.bak
try {
	Write-Host '==> go mod: replacing raylib with the Raylib-Go-Wasm fork'
	Invoke-Go 'mod' 'edit' `
		'-replace' 'github.com/gen2brain/raylib-go/raylib=./Raylib-Go-Wasm/raylib' `
		'-replace' 'github.com/BrownNPC/Raylib-Go-Wasm/wasm-runtime=./Raylib-Go-Wasm/wasm-runtime'
	Invoke-Go 'mod' 'tidy'

	Write-Host '==> building wasm'
	$env:GOOS = 'js'
	$env:GOARCH = 'wasm'
	Invoke-Go 'build' '-o' './Raylib-Go-Wasm/index/main.wasm' '.'

	Write-Host '==> copying wasm_exec.js from GOROOT'
	Copy-Item "$(go env GOROOT)/lib/wasm/wasm_exec.js" ./Raylib-Go-Wasm/index/wasm_exec.js

	if (-not (Test-Path server.exe)) {
		Write-Host '==> building dev server (first time only)'
		Invoke-Go 'build' '-o' 'server.exe' './Raylib-Go-Wasm/server/server.go'
	}
}
finally {
	Move-Item go.mod.wasm.bak go.mod -Force
	Move-Item go.sum.wasm.bak go.sum -Force
	Remove-Item Env:GOOS, Env:GOARCH -ErrorAction SilentlyContinue
}

Write-Host ''
Write-Host 'Done. Start the app with:  ./server.exe'
Write-Host 'Then open:                 http://localhost:8080'
