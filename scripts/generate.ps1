#Requires -Version 5.1
<#
.SYNOPSIS
    Генерирует gRPC-стабы из proto/signer/signer.proto в gen/signer/.
.DESCRIPTION
    - Проверяет наличие protoc, protoc-gen-go, protoc-gen-go-grpc.
    - Устанавливает Go-плагины через `go install`, если они отсутствуют.
    - Запускает protoc с нужными флагами.
    - Завершается с ненулевым кодом при любой ошибке.
#>

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

# --- Константы ---
$RepoRoot  = Split-Path -Parent $PSScriptRoot
$ProtoFile = "$RepoRoot\proto\signer\signer.proto"
$ProtoDir  = "$RepoRoot\proto"
$OutDir    = "$RepoRoot\gen\signer"

# --- Вспомогательные функции ---
function Assert-Command {
    param([string]$Name, [string]$InstallHint)
    if (-not (Get-Command $Name -ErrorAction SilentlyContinue)) {
        Write-Error "Команда '$Name' не найдена. $InstallHint"
    }
}

function Install-GoTool {
    param([string]$Package, [string]$Binary)
    if (-not (Get-Command $Binary -ErrorAction SilentlyContinue)) {
        Write-Host "Устанавливаю $Binary..."
        go install $Package
        # Обновляем PATH для текущей сессии
        $gobin = go env GOPATH
        $env:PATH = "$gobin\bin;$env:PATH"
        if (-not (Get-Command $Binary -ErrorAction SilentlyContinue)) {
            Write-Error "Не удалось найти '$Binary' после установки. Убедитесь, что GOPATH\bin в PATH."
        }
        Write-Host "$Binary успешно установлен."
    } else {
        Write-Host "$Binary уже установлен."
    }
}

# --- Проверки ---
Assert-Command 'protoc' `
    'Установите Protocol Buffers compiler: https://github.com/protocolbuffers/protobuf/releases'

Assert-Command 'go' `
    'Установите Go: https://golang.org/dl/'

# --- Go-плагины ---
Install-GoTool 'google.golang.org/protobuf/cmd/protoc-gen-go@latest'      'protoc-gen-go'
Install-GoTool 'google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest'     'protoc-gen-go-grpc'

# --- Создаём выходную директорию ---
if (-not (Test-Path $OutDir)) {
    New-Item -ItemType Directory -Path $OutDir | Out-Null
    Write-Host "Создана директория: $OutDir"
}

# --- Генерация ---
Write-Host "Запускаю protoc..."

protoc `
    --proto_path="$ProtoDir" `
    --go_out="$RepoRoot\gen" `
    --go_opt=paths=source_relative `
    --go-grpc_out="$RepoRoot\gen" `
    --go-grpc_opt=paths=source_relative `
    "$ProtoFile"

if ($LASTEXITCODE -ne 0) {
    Write-Error "protoc завершился с ошибкой (код $LASTEXITCODE)."
}

Write-Host "Генерация завершена. Файлы в: $OutDir"
