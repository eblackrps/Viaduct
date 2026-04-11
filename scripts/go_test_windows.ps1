param(
    [Parameter(ValueFromRemainingArguments = $true)]
    [string[]]$GoTestArgs
)

$ErrorActionPreference = "Stop"

function Convert-GoTestArguments {
    param(
        [Parameter(Mandatory = $true)]
        [string[]]$Arguments
    )

    $packagePatterns = New-Object System.Collections.Generic.List[string]
    $runtimeFlags = New-Object System.Collections.Generic.List[string]
    $coverageProfile = "coverage.out"

    for ($index = 0; $index -lt $Arguments.Count; $index++) {
        $argument = $Arguments[$index]
        if (-not $argument.StartsWith("-")) {
            $packagePatterns.Add($argument)
            continue
        }

        switch -Regex ($argument) {
            '^-coverprofile=(.+)$' {
                $coverageProfile = $Matches[1]
                continue
            }
            '^-coverprofile$' {
                $index++
                $coverageProfile = $Arguments[$index]
                continue
            }
            '^-v$' {
                $runtimeFlags.Add("-test.v")
                continue
            }
            '^-count=(.+)$' {
                $runtimeFlags.Add("-test.count=$($Matches[1])")
                continue
            }
            '^-count$' {
                $index++
                $runtimeFlags.Add("-test.count=$($Arguments[$index])")
                continue
            }
            '^-run=(.+)$' {
                $runtimeFlags.Add("-test.run=$($Matches[1])")
                continue
            }
            '^-run$' {
                $index++
                $runtimeFlags.Add("-test.run=$($Arguments[$index])")
                continue
            }
            '^-timeout=(.+)$' {
                $runtimeFlags.Add("-test.timeout=$($Matches[1])")
                continue
            }
            '^-timeout$' {
                $index++
                $runtimeFlags.Add("-test.timeout=$($Arguments[$index])")
                continue
            }
            '^-short$' {
                $runtimeFlags.Add("-test.short")
                continue
            }
        }
    }

    if ($packagePatterns.Count -eq 0) {
        $packagePatterns.Add("./...")
    }
    if (-not ($runtimeFlags | Where-Object { $_ -like "-test.timeout=*" })) {
        $runtimeFlags.Add("-test.timeout=10m0s")
    }

    return @{
        PackagePatterns = [string[]]$packagePatterns
        RuntimeFlags    = [string[]]$runtimeFlags
        CoverageProfile = $coverageProfile
    }
}

function Get-TestPackages {
    param(
        [Parameter(Mandatory = $true)]
        [string]$GoBinary,
        [Parameter(Mandatory = $true)]
        [string[]]$Patterns
    )

    $packages = & $GoBinary list -f '{{if or .TestGoFiles .XTestGoFiles}}{{.ImportPath}}|{{.Dir}}{{end}}' @Patterns
    if ($LASTEXITCODE -ne 0) {
        exit $LASTEXITCODE
    }

    return $packages |
        ForEach-Object { $_.Trim() } |
        Where-Object { $_ -ne "" } |
        ForEach-Object {
            $parts = $_ -split '\|', 2
            [pscustomobject]@{
                ImportPath = $parts[0]
                Dir        = $parts[1]
            }
        }
}

function Invoke-TestBinary {
    param(
        [Parameter(Mandatory = $true)]
        [string]$BinaryPath,
        [Parameter(Mandatory = $true)]
        [string[]]$RuntimeFlags
    )

    $attempts = 6
    for ($attempt = 1; $attempt -le $attempts; $attempt++) {
        try {
            & $BinaryPath @RuntimeFlags | Out-Host
            return $LASTEXITCODE
        }
        catch {
            $message = $_.Exception.Message
            $blocked = $message -like "*Application Control policy has blocked this file*" -or $message -like "*Device Guard policy*"
            if (-not $blocked -or $attempt -eq $attempts) {
                throw
            }

            Start-Sleep -Seconds 5
        }
    }

    return 1
}

function Run-PackageBinary {
    param(
        [Parameter(Mandatory = $true)]
        [string]$BinaryPath,
        [Parameter(Mandatory = $true)]
        [string]$WorkingDirectory,
        [Parameter(Mandatory = $true)]
        [string[]]$RuntimeFlags
    )

    Push-Location $WorkingDirectory
    try {
        return Invoke-TestBinary -BinaryPath $BinaryPath -RuntimeFlags $RuntimeFlags
    }
    finally {
        Pop-Location
    }
}

$workspaceRoot = [System.IO.Path]::GetFullPath((Join-Path $PSScriptRoot ".."))
$tempRoot = [Environment]::GetEnvironmentVariable("VIADUCT_WINDOWS_TEST_BASE")
if (-not $tempRoot) {
    $tempRoot = Join-Path $workspaceRoot ".tmp\test"
}
$binaryRoot = Join-Path $workspaceRoot ".tmp\exec"
$coverRoot = Join-Path $tempRoot "cover"
New-Item -ItemType Directory -Force -Path $tempRoot, $binaryRoot, $coverRoot | Out-Null

$goBinary = (Get-Command go -ErrorAction Stop).Source
$parsed = Convert-GoTestArguments -Arguments $GoTestArgs
$packages = Get-TestPackages -GoBinary $goBinary -Patterns $parsed.PackagePatterns
$packageIndex = 0
$compiledPackages = New-Object System.Collections.Generic.List[object]
$coverageSkipped = New-Object System.Collections.Generic.List[string]

if (Test-Path -LiteralPath $coverRoot) {
    Get-ChildItem -LiteralPath $coverRoot -Force | Remove-Item -Recurse -Force
}
New-Item -ItemType Directory -Force -Path $coverRoot | Out-Null

foreach ($package in $packages) {
    $binaryName = ("pkg-{0:D3}.exe" -f $packageIndex)
    $packageIndex++
    $binaryPath = Join-Path $binaryRoot $binaryName
    if (Test-Path -LiteralPath $binaryPath) {
        Remove-Item -LiteralPath $binaryPath -Force
    }

    & $goBinary test -c -cover -o $binaryPath $package.ImportPath
    if ($LASTEXITCODE -ne 0) {
        exit $LASTEXITCODE
    }

    $compiledPackages.Add([pscustomobject]@{
        Package    = $package
        BinaryPath = $binaryPath
    })
}

foreach ($compiled in $compiledPackages) {
    $package = $compiled.Package
    $binaryPath = $compiled.BinaryPath
    $runtimeArgs = @("-test.gocoverdir=$coverRoot") + $parsed.RuntimeFlags
    $stopwatch = [System.Diagnostics.Stopwatch]::StartNew()
    try {
        try {
            $exitCode = Run-PackageBinary -BinaryPath $binaryPath -WorkingDirectory $package.Dir -RuntimeFlags $runtimeArgs
        }
        catch {
            $message = $_.Exception.Message
            $blocked = $message -like "*Application Control policy has blocked this file*" -or $message -like "*Device Guard policy*"
            if (-not $blocked) {
                throw
            }

            Remove-Item -LiteralPath $binaryPath -Force -ErrorAction SilentlyContinue
            & $goBinary test -c -o $binaryPath $package.ImportPath
            if ($LASTEXITCODE -ne 0) {
                exit $LASTEXITCODE
            }

            Write-Host ("warn`t{0}`tcoverage binary blocked by Application Control; rerunning without coverage" -f $package.ImportPath)
            $coverageSkipped.Add($package.ImportPath)
            $exitCode = Run-PackageBinary -BinaryPath $binaryPath -WorkingDirectory $package.Dir -RuntimeFlags $parsed.RuntimeFlags
        }
    }
    finally {
        $stopwatch.Stop()
        Remove-Item -LiteralPath $binaryPath -Force -ErrorAction SilentlyContinue
    }

    if ($exitCode -ne 0) {
        Write-Host ("FAIL`t{0}`t{1:N3}s" -f $package.ImportPath, $stopwatch.Elapsed.TotalSeconds)
        exit $exitCode
    }

    Write-Host ("ok`t{0}`t{1:N3}s" -f $package.ImportPath, $stopwatch.Elapsed.TotalSeconds)
}

& $goBinary tool covdata textfmt -i $coverRoot -o $parsed.CoverageProfile
if ($coverageSkipped.Count -gt 0) {
    Write-Host ("warn`tskipped coverage aggregation for: {0}" -f ($coverageSkipped -join ", "))
}
exit $LASTEXITCODE
