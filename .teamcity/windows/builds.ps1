Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
$ProgressPreference = "SilentlyContinue"

# Relative path to working directory
$TunnellinkDirectory = "go\src\github.com\khulnasoft\tunnellink"

cd $TunnellinkDirectory

Write-Output "Building for amd64"
$env:TARGET_OS = "windows"
$env:CGO_ENABLED = 1
$env:TARGET_ARCH = "amd64"
$env:Path = "$Env:Temp\go\bin;$($env:Path)"

go env
go version

& make tunnellink
if ($LASTEXITCODE -ne 0) { throw "Failed to build tunnellink for amd64" }
copy .\tunnellink.exe .\tunnellink-windows-amd64.exe

Write-Output "Building for 386"
$env:CGO_ENABLED = 0
$env:TARGET_ARCH = "386"
make tunnellink
if ($LASTEXITCODE -ne 0) { throw "Failed to build tunnellink for 386" }
copy .\tunnellink.exe .\tunnellink-windows-386.exe