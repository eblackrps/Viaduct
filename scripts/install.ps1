param(
    [string]$SourceBin = "bin/viaduct.exe",
    [string]$SourceWeb = "web",
    [string]$Prefix = "$env:LOCALAPPDATA\\Programs\\Viaduct"
)

$ErrorActionPreference = "Stop"

if (-not (Test-Path -LiteralPath $SourceBin)) {
    throw "viaduct install: binary not found at $SourceBin"
}

$binDir = Join-Path $Prefix "bin"
$shareDir = Join-Path $Prefix "share"
$webTarget = Join-Path $shareDir "web"

New-Item -ItemType Directory -Force -Path $binDir | Out-Null
New-Item -ItemType Directory -Force -Path $shareDir | Out-Null

Copy-Item -LiteralPath $SourceBin -Destination (Join-Path $binDir "viaduct.exe") -Force

if (Test-Path -LiteralPath $SourceWeb) {
    if (Test-Path -LiteralPath $webTarget) {
        Remove-Item -LiteralPath $webTarget -Recurse -Force
    }
    Copy-Item -LiteralPath $SourceWeb -Destination $webTarget -Recurse -Force
}

Write-Host "Installed Viaduct to $Prefix"
