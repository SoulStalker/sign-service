#Requires -Version 5.1
#Requires -RunAsAdministrator
<#
.SYNOPSIS
    Устанавливает sign-service как Windows-службу через NSSM.
.PARAMETER ConfigPath
    Путь к YAML-конфигу. По умолчанию: config\prod.yml рядом с exe.
.PARAMETER ServiceName
    Имя Windows-службы. По умолчанию: sign-service.
.EXAMPLE
    .\install-service.ps1
    .\install-service.ps1 -ConfigPath C:\sign-service\config\prod.yml -ServiceName sign-service
#>
param(
    [string]$ServiceName = 'sign-service',
    [string]$ConfigPath  = ''
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

# --- Пути ---
$RepoRoot  = Split-Path -Parent $PSScriptRoot
$ExePath   = "$RepoRoot\sign-service.exe"
$AppDir    = $RepoRoot

if ($ConfigPath -eq '') {
    $ConfigPath = "$RepoRoot\config\prod.yml"
}

# --- Проверки ---
if (-not (Get-Command 'nssm' -ErrorAction SilentlyContinue)) {
    Write-Error @"
NSSM не найден. Установите его одним из способов:
  winget install NSSM.NSSM
  choco install nssm
  Или скачайте вручную: https://nssm.cc/download
"@
}

if (-not (Test-Path $ExePath)) {
    Write-Error "Исполняемый файл не найден: $ExePath`nСначала выполните: go build -o sign-service.exe .\cmd\sign-service"
}

if (-not (Test-Path $ConfigPath)) {
    Write-Error "Конфиг не найден: $ConfigPath"
}

# --- Удаляем старую установку, если есть ---
$existing = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
if ($existing) {
    Write-Host "Служба '$ServiceName' уже существует — останавливаем и удаляем..."
    if ($existing.Status -eq 'Running') {
        nssm stop $ServiceName confirm
    }
    nssm remove $ServiceName confirm
}

# --- Установка ---
Write-Host "Устанавливаем службу '$ServiceName'..."

nssm install $ServiceName $ExePath "--config" $ConfigPath

nssm set $ServiceName AppDirectory    $AppDir
nssm set $ServiceName DisplayName     'Sign Service (GOST Crypto)'
nssm set $ServiceName Description     'gRPC-сервис криптографической подписи через Windows Certificate Store'
nssm set $ServiceName Start           SERVICE_AUTO_START
nssm set $ServiceName AppStdout       "$AppDir\logs\service.log"
nssm set $ServiceName AppStderr       "$AppDir\logs\service.log"
nssm set $ServiceName AppRotateFiles  1
nssm set $ServiceName AppRotateBytes  10485760  # 10 МБ

# --- Создаём директорию для логов ---
$logsDir = "$AppDir\logs"
if (-not (Test-Path $logsDir)) {
    New-Item -ItemType Directory -Path $logsDir | Out-Null
}

# --- Запуск ---
Write-Host "Запускаем службу '$ServiceName'..."
nssm start $ServiceName

$svc = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
if ($svc -and $svc.Status -eq 'Running') {
    Write-Host "Служба '$ServiceName' успешно запущена."
} else {
    Write-Warning "Служба установлена, но не запущена. Проверьте логи: $logsDir\service.log"
}
