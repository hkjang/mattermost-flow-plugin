$ErrorActionPreference = 'Stop'

$repoRoot = Split-Path -Parent $PSScriptRoot
Set-Location $repoRoot

function Resolve-CommandPath {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Name
    )

    $command = Get-Command $Name -ErrorAction SilentlyContinue
    if ($command) {
        return $command.Source
    }

    throw "Unable to locate required command: $Name"
}

$go = Resolve-CommandPath 'go'
$npm = Resolve-CommandPath 'npm.cmd'

$hash = (& git rev-parse --short HEAD).Trim()
$tagLatest = (& git describe --tags --match 'v*' --abbrev=0).Trim()
$tagCurrent = ((& git tag --points-at HEAD) | Out-String).Trim()

New-Item -ItemType Directory -Force build\bin | Out-Null

$ldflags = @(
    "-X `"main.BuildHashShort=$hash`"",
    "-X `"main.BuildTagLatest=$tagLatest`"",
    "-X `"main.BuildTagCurrent=$tagCurrent`""
) -join ' '

& $go build "-ldflags=$ldflags" -o build\bin\manifest.exe .\build\manifest
& .\build\bin\manifest.exe apply

Remove-Item server\dist -Recurse -Force -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Force server\dist | Out-Null

$targets = @(
    @{GOOS = 'linux'; GOARCH = 'amd64'; Output = 'plugin-linux-amd64'},
    @{GOOS = 'linux'; GOARCH = 'arm64'; Output = 'plugin-linux-arm64'},
    @{GOOS = 'darwin'; GOARCH = 'amd64'; Output = 'plugin-darwin-amd64'},
    @{GOOS = 'darwin'; GOARCH = 'arm64'; Output = 'plugin-darwin-arm64'},
    @{GOOS = 'windows'; GOARCH = 'amd64'; Output = 'plugin-windows-amd64.exe'}
)

foreach ($target in $targets) {
    $env:CGO_ENABLED = '0'
    $env:GOOS = $target.GOOS
    $env:GOARCH = $target.GOARCH
    & $go build -trimpath -o (Join-Path server\dist $target.Output) .\server
}

Remove-Item Env:GOOS -ErrorAction SilentlyContinue
Remove-Item Env:GOARCH -ErrorAction SilentlyContinue

& $npm run build --prefix webapp

$pluginID = (& .\build\bin\manifest.exe id).Trim()
$pluginVersion = (& .\build\bin\manifest.exe version).Trim()
$bundleName = "$pluginID-$pluginVersion.tar.gz"

Remove-Item dist -Recurse -Force -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Force (Join-Path dist $pluginID) | Out-Null

& .\build\bin\manifest.exe dist

if (Test-Path assets) {
    Copy-Item assets -Destination (Join-Path dist $pluginID) -Recurse
}
if (Test-Path public) {
    Copy-Item public -Destination (Join-Path dist $pluginID) -Recurse
}

New-Item -ItemType Directory -Force (Join-Path dist "$pluginID\server") | Out-Null
Copy-Item server\dist -Destination (Join-Path dist "$pluginID\server") -Recurse

New-Item -ItemType Directory -Force (Join-Path dist "$pluginID\webapp") | Out-Null
Copy-Item webapp\dist -Destination (Join-Path dist "$pluginID\webapp") -Recurse

& .\build\bin\manifest.exe bundle $pluginID $bundleName

Write-Output "BundlePath=$(Join-Path dist $bundleName)"
