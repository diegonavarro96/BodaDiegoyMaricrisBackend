# Cross-compile the RSVP server for the AWS Linux box from Windows.
# Output: ./wedding-rsvp (linux/amd64). Strip debug info, no CGO.

$ErrorActionPreference = 'Stop'

Push-Location $PSScriptRoot/..
try {
    $env:GOOS = 'linux'
    $env:GOARCH = 'amd64'
    $env:CGO_ENABLED = '0'
    go build -trimpath -ldflags '-s -w' -o wedding-rsvp .
    Write-Host "built: $(Resolve-Path wedding-rsvp)"
}
finally {
    Pop-Location
    Remove-Item Env:GOOS, Env:GOARCH, Env:CGO_ENABLED -ErrorAction SilentlyContinue
}
