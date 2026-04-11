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

function Convert-GoTestArguments {
    param(
        [Parameter(Mandatory = $true)]
        [string[]]$Arguments
    )

    $packagePatterns = New-Object System.Collections.Generic.List[string]
    $buildFlags = New-Object System.Collections.Generic.List[string]
    $runtimeFlags = New-Object System.Collections.Generic.List[string]

    for ($index = 0; $index -lt $Arguments.Count; $index++) {
        $argument = $Arguments[$index]
        if (-not $argument.StartsWith("-")) {
            $packagePatterns.Add($argument)
            continue
        }

        switch -Regex ($argument) {
            '^-race$' {
                $buildFlags.Add($argument)
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
            '^-shuffle=(.+)$' {
                $runtimeFlags.Add("-test.shuffle=$($Matches[1])")
                continue
            }
            '^-shuffle$' {
                $index++
                $runtimeFlags.Add("-test.shuffle=$($Arguments[$index])")
                continue
            }
            '^-bench=(.+)$' {
                $runtimeFlags.Add("-test.bench=$($Matches[1])")
                continue
            }
            '^-bench$' {
                $index++
                $runtimeFlags.Add("-test.bench=$($Arguments[$index])")
                continue
            }
            default {
                $buildFlags.Add($argument)
                if ($argument -in @('-tags', '-mod', '-modfile')) {
                    $index++
                    $buildFlags.Add($Arguments[$index])
                }
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
        BuildFlags      = [string[]]$buildFlags
        RuntimeFlags    = [string[]]$runtimeFlags
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

$gcc = Resolve-CompilerPath -EnvironmentVariable "VIADUCT_WINDOWS_RACE_CC" -BinaryName "x86_64-w64-mingw32-gcc.exe"
$gxx = Resolve-CompilerPath -EnvironmentVariable "VIADUCT_WINDOWS_RACE_CXX" -BinaryName "x86_64-w64-mingw32-g++.exe"

$workspaceRoot = [System.IO.Path]::GetFullPath((Join-Path $PSScriptRoot ".."))
$binaryRoot = Join-Path $workspaceRoot ".tmp\exec"
New-Item -ItemType Directory -Force -Path $binaryRoot | Out-Null

$env:CGO_ENABLED = "1"
$env:CC = $gcc
$env:CXX = $gxx

$goBinary = (Get-Command go -ErrorAction Stop).Source
$parsed = Convert-GoTestArguments -Arguments $GoTestArgs
$nonRaceBuildFlags = @($parsed.BuildFlags | Where-Object { $_ -ne "-race" })
$packages = Get-TestPackages -GoBinary $goBinary -Patterns $parsed.PackagePatterns
$packageIndex = 0
$compiledPackages = New-Object System.Collections.Generic.List[object]

foreach ($package in $packages) {
    $binaryName = ("pkg-{0:D3}.exe" -f $packageIndex)
    $packageIndex++
    $binaryPath = Join-Path $binaryRoot $binaryName
    if (Test-Path -LiteralPath $binaryPath) {
        Remove-Item -LiteralPath $binaryPath -Force
    }

    $compileArgs = @("test", "-c") + $parsed.BuildFlags + @("-o", $binaryPath, $package.ImportPath)
    & $goBinary @compileArgs
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
    $stopwatch = [System.Diagnostics.Stopwatch]::StartNew()
    try {
        try {
            $exitCode = Run-PackageBinary -BinaryPath $binaryPath -WorkingDirectory $package.Dir -RuntimeFlags $parsed.RuntimeFlags
        }
        catch {
            $message = $_.Exception.Message
            $blocked = $message -like "*Application Control policy has blocked this file*" -or $message -like "*Device Guard policy*"
            if (-not $blocked) {
                throw
            }

            Remove-Item -LiteralPath $binaryPath -Force -ErrorAction SilentlyContinue
            $fallbackArgs = @("test", "-c") + $nonRaceBuildFlags + @("-o", $binaryPath, $package.ImportPath)
            & $goBinary @fallbackArgs
            if ($LASTEXITCODE -ne 0) {
                exit $LASTEXITCODE
            }

            Write-Host ("warn`t{0}`trace binary blocked by Application Control; rerunning without -race" -f $package.ImportPath)
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

exit 0
