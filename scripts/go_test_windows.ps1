param(
    [Parameter(ValueFromRemainingArguments = $true)]
    [string[]]$GoTestArgs
)

$ErrorActionPreference = "Stop"

$base = Join-Path $env:LOCALAPPDATA "ViaductTest"
$gocache = Join-Path $base "gocache"
$gotmp = Join-Path $base "gotmp"
$gomodcache = Join-Path $base "gomodcache"
New-Item -ItemType Directory -Force -Path $base, $gocache, $gotmp, $gomodcache | Out-Null

$env:GOCACHE = $gocache
$env:GOTMPDIR = $gotmp
$env:GOMODCACHE = $gomodcache
$env:TMP = $base
$env:TEMP = $base

$goBinary = (Get-Command go -ErrorAction Stop).Source
& $goBinary test @GoTestArgs
exit $LASTEXITCODE
