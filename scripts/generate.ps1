#Requires -Version 5.1
<#
.SYNOPSIS
    Generates gRPC stubs from proto/signer/signer.proto into gen/signer/.
.DESCRIPTION
    - Checks for protoc, protoc-gen-go, protoc-gen-go-grpc.
    - Installs Go plugins via `go install` if missing.
    - Runs protoc with required flags.
    - Exits with non-zero code on any error.
#>

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

# --- Constants ---
$RepoRoot  = Split-Path -Parent $PSScriptRoot
$ProtoFile = "$RepoRoot\proto\signer\signer.proto"
$ProtoDir  = "$RepoRoot\proto"
$OutDir    = "$RepoRoot\gen\signer"

# --- Helper functions ---
function Assert-Command {
    param([string]$Name, [string]$InstallHint)
    if (-not (Get-Command $Name -ErrorAction SilentlyContinue)) {
        Write-Error "Command '$Name' not found. $InstallHint"
    }
}

function Install-GoTool {
    param([string]$Package, [string]$Binary)
    if (-not (Get-Command $Binary -ErrorAction SilentlyContinue)) {
        Write-Host "Installing $Binary..."
        go install $Package
        # Update PATH for current session
        $gobin = go env GOPATH
        $env:PATH = "$gobin\bin;$env:PATH"
        if (-not (Get-Command $Binary -ErrorAction SilentlyContinue)) {
            Write-Error "Could not find '$Binary' after installation. Make sure GOPATH\bin is in PATH."
        }
        Write-Host "$Binary installed successfully."
    } else {
        Write-Host "$Binary is already installed."
    }
}

# --- Prerequisites ---
Assert-Command 'protoc' `
    'Install Protocol Buffers compiler: https://github.com/protocolbuffers/protobuf/releases'

Assert-Command 'go' `
    'Install Go: https://golang.org/dl/'

# --- Go plugins ---
Install-GoTool 'google.golang.org/protobuf/cmd/protoc-gen-go@latest'      'protoc-gen-go'
Install-GoTool 'google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest'     'protoc-gen-go-grpc'

# --- Create output directory ---
if (-not (Test-Path $OutDir)) {
    New-Item -ItemType Directory -Path $OutDir | Out-Null
    Write-Host "Created directory: $OutDir"
}

# --- Generate ---
Write-Host "Running protoc..."

protoc `
    --proto_path="$ProtoDir" `
    --go_out="$RepoRoot\gen" `
    --go_opt=paths=source_relative `
    --go-grpc_out="$RepoRoot\gen" `
    --go-grpc_opt=paths=source_relative `
    "$ProtoFile"

if ($LASTEXITCODE -ne 0) {
    Write-Error "protoc failed with exit code $LASTEXITCODE."
}

Write-Host "Generation complete. Files written to: $OutDir"
