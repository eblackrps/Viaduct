param(
    [Parameter(ValueFromRemainingArguments = $true)]
    [string[]]$GoTestArgs
)

$ErrorActionPreference = "Stop"

function Resolve-CompilerPath {
    param(
        [Parameter(Mandatory = $true)]
        [string]$EnvironmentVariable,
        [Parameter(Mandatory = $true)]
        [string]$BinaryName
    )

    $override = [Environment]::GetEnvironmentVariable($EnvironmentVariable)
    if ($override) {
        if (-not (Test-Path -LiteralPath $override)) {
            throw "Configured $EnvironmentVariable path '$override' does not exist."
        }

        return $override
    }

    $packagesRoot = Join-Path $env:LOCALAPPDATA "Microsoft\WinGet\Packages"
    $packageRoot = Get-ChildItem -LiteralPath $packagesRoot -Directory |
        Where-Object { $_.Name -like "MartinStorsjo.LLVM-MinGW.UCRT*" } |
        Sort-Object Name -Descending |
        Select-Object -First 1
    if (-not $packageRoot) {
        throw "Install the LLVM-MinGW UCRT toolchain with 'winget install --id MartinStorsjo.LLVM-MinGW.UCRT -e' before running race tests on Windows."
    }

    $toolchainRoot = Get-ChildItem -LiteralPath $packageRoot.FullName -Directory |
        Where-Object { $_.Name -like "llvm-mingw-*" } |
        Sort-Object Name -Descending |
        Select-Object -First 1
    if (-not $toolchainRoot) {
        throw "Unable to locate the installed LLVM-MinGW toolchain under '$($packageRoot.FullName)'."
    }

    $binaryPath = Join-Path $toolchainRoot.FullName ("bin\" + $BinaryName)
    if (-not (Test-Path -LiteralPath $binaryPath)) {
        throw "Unable to locate '$BinaryName' under '$($toolchainRoot.FullName)\bin'."
    }

    return $binaryPath
}

$gcc = Resolve-CompilerPath -EnvironmentVariable "VIADUCT_WINDOWS_RACE_CC" -BinaryName "x86_64-w64-mingw32-gcc.exe"
$gxx = Resolve-CompilerPath -EnvironmentVariable "VIADUCT_WINDOWS_RACE_CXX" -BinaryName "x86_64-w64-mingw32-g++.exe"

$base = Join-Path $env:LOCALAPPDATA "ViaductRace"
$gocache = Join-Path $base "gocache"
$gotmp = Join-Path $base "gotmp"
$gomodcache = Join-Path $base "gomodcache"
New-Item -ItemType Directory -Force -Path $base, $gocache, $gotmp, $gomodcache | Out-Null

$env:CGO_ENABLED = "1"
$env:CC = $gcc
$env:CXX = $gxx
$env:GOCACHE = $gocache
$env:GOTMPDIR = $gotmp
$env:GOMODCACHE = $gomodcache
$env:TMP = $base
$env:TEMP = $base

$goBinary = (Get-Command go -ErrorAction Stop).Source
& $goBinary test @GoTestArgs
exit $LASTEXITCODE
