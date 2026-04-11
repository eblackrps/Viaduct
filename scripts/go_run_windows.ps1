[CmdletBinding(PositionalBinding = $false)]
param(
    [string]$Ldflags = "",

    [Parameter(ValueFromRemainingArguments = $true)]
    [string[]]$GoRunArgs
)

$ErrorActionPreference = "Stop"

$base = Join-Path $env:LOCALAPPDATA "ViaductRun"
$gocache = Join-Path $base "gocache"
$gotmp = Join-Path $base "gotmp"
$gomodcache = Join-Path $base "gomodcache"
New-Item -ItemType Directory -Force -Path $base, $gocache, $gotmp, $gomodcache | Out-Null

$env:GOCACHE = $gocache
$env:GOTMPDIR = $gotmp
$env:GOMODCACHE = $gomodcache
$env:TMP = $base
$env:TEMP = $base

function Invoke-Executable {
    param(
        [Parameter(Mandatory = $true)]
        [string]$BinaryPath,
        [Parameter(Mandatory = $true)]
        [string[]]$Arguments
    )

    $attempts = 4
    for ($attempt = 1; $attempt -le $attempts; $attempt++) {
        try {
            & $BinaryPath @Arguments
            return $LASTEXITCODE
        }
        catch {
            $message = $_.Exception.Message
            $blocked = $message -like "*Application Control policy has blocked this file*" -or $message -like "*Device Guard policy*"
            if (-not $blocked -or $attempt -eq $attempts) {
                throw
            }

            Start-Sleep -Seconds 3
        }
    }

    return 1
}

$goBinary = (Get-Command go -ErrorAction Stop).Source
$goArgs = @("run")
if ($Ldflags -ne "") {
    $goArgs += "-ldflags"
    $goArgs += $Ldflags
}
$goArgs += $GoRunArgs

$hadNativePreference = Test-Path Variable:PSNativeCommandUseErrorActionPreference
if ($hadNativePreference) {
    $previousNativePreference = $PSNativeCommandUseErrorActionPreference
    $script:PSNativeCommandUseErrorActionPreference = $false
}
$previousErrorActionPreference = $ErrorActionPreference
$ErrorActionPreference = "Continue"

try {
    $runOutput = & $goBinary @goArgs 2>&1
    $runExitCode = $LASTEXITCODE
}
finally {
    $ErrorActionPreference = $previousErrorActionPreference
    if ($hadNativePreference) {
        $script:PSNativeCommandUseErrorActionPreference = $previousNativePreference
    }
}

foreach ($line in $runOutput) {
    if ($line -is [System.Management.Automation.ErrorRecord]) {
        $line.Exception.Message | Out-Host
        continue
    }

    $line | Out-Host
}

if ($runExitCode -eq 0) {
    exit 0
}

$message = ($runOutput | Out-String)
$blocked = $message -like "*Application Control policy has blocked this file*" -or $message -like "*Device Guard policy*"
if (-not $blocked) {
    exit $runExitCode
}

if ($GoRunArgs.Count -eq 0) {
    exit $runExitCode
}

$target = $GoRunArgs[0]
$execArgs = @()
if ($GoRunArgs.Count -gt 1) {
    $execArgs = $GoRunArgs[1..($GoRunArgs.Count - 1)]
}

$binDir = Join-Path $base "bin"
New-Item -ItemType Directory -Force -Path $binDir | Out-Null
$fallbackBinary = Join-Path $binDir ("run-" + [Guid]::NewGuid().ToString("N") + ".exe")

try {
    $buildArgs = @("build")
    if ($Ldflags -ne "") {
        $buildArgs += "-ldflags"
        $buildArgs += $Ldflags
    }
    $buildArgs += "-o"
    $buildArgs += $fallbackBinary
    $buildArgs += $target

    & $goBinary @buildArgs
    if ($LASTEXITCODE -ne 0) {
        exit $LASTEXITCODE
    }

    $exitCode = Invoke-Executable -BinaryPath $fallbackBinary -Arguments $execArgs
    exit $exitCode
}
finally {
    Remove-Item -LiteralPath $fallbackBinary -Force -ErrorAction SilentlyContinue
}
