# EnetSender Go -- deploy script
#
# -- Build approach -----------------------------------------------------------
# PC cross-compiles the Go binary, builds a linux/arm64 Docker image (FROM scratch,
# ~6 MB), and pushes it to ghcr.io. The HA supervisor pulls the pre-built image on
# `ha apps update` -- reducing deploy time from ~15 minutes to ~15 seconds.
#
# -- Why image: field in config.yaml ------------------------------------------
# When config.yaml contains `image: ghcr.io/joergni/enetsender`, the supervisor
# skips the local Dockerfile build entirely and does a plain `docker pull` using
# the image name + version as the tag. No Dockerfile or binary needs to be on the Pi.
#
# -- Why ghcr.io --------------------------------------------------------------
# Free, no Pi infrastructure required, works the same as official HA store add-ons.
# Package must be set to public so the Pi can pull without credentials.
#
# -- Why ha store reload before ha apps update --------------------------------
# The supervisor caches add-on metadata. Without store reload it may not notice
# the new version in config.yaml. store reload occasionally times out with
# "context deadline exceeded" -- transient and not fatal; update proceeds anyway.
#
# -- Why watchdog is disabled before update -----------------------------------
# The watchdog fires the moment the supervisor stops the running container,
# creating a competing restart job that causes "Another job is running" errors.
# Disabled before update, re-enabled after the add-on reaches started state.
#
# -- Git commit message behavior ----------------------------------------------
# This script requires -CommitMessage. There is no interactive prompt.
# AI agents must pass -CommitMessage explicitly, using a concise summary of the
# code they changed, e.g. .\deploy_enet.ps1 -CommitMessage "Fix scheduler drift".

param(
    [string]$CommitMessage
)

$ErrorActionPreference = "Stop"
$RepoRoot = Split-Path -Parent $PSCommandPath
Set-Location $RepoRoot

if ([string]::IsNullOrWhiteSpace($CommitMessage)) {
    throw "Missing required -CommitMessage. Usage: .\deploy_enet.ps1 -CommitMessage 'Fix scheduler drift'"
}

$ConfigPath = Join-Path $RepoRoot "config.yaml"
$BinaryPath = Join-Path $RepoRoot "enetsender"
$Slug = "local_enetsender"
$Remote = "/addons/local/enetsender"
$Image = "ghcr.io/joergni/enetsender"

function GetGitRepoRoot([string]$Path) {
    $root = & git -C $Path rev-parse --show-toplevel 2>$null
    if ($LASTEXITCODE -ne 0) { throw "git repository not found" }
    return ($root | Select-Object -First 1).Trim()
}

function TestHasPendingGitChanges([string]$RepoPath) {
    $status = & git -C $RepoPath status --porcelain
    if ($LASTEXITCODE -ne 0) { throw "git status failed" }
    return [bool]($status | Select-Object -First 1)
}

function PushToGitHub([string]$RepoPath, [string]$CommitMessage) {
    $resolvedRepoRoot = GetGitRepoRoot $RepoPath

    Push-Location $resolvedRepoRoot
    try {
        & git add -A
        if ($LASTEXITCODE -ne 0) { throw "git add failed" }

        & git diff --cached --quiet
        if ($LASTEXITCODE -gt 1) { throw "git diff --cached failed" }
        if ($LASTEXITCODE -eq 0) {
            Write-Host "  no git changes to commit"
            return
        }

        & git commit -m $CommitMessage
        if ($LASTEXITCODE -ne 0) { throw "git commit failed" }

        & git push
        if ($LASTEXITCODE -ne 0) { throw "git push failed" }

        Write-Host "  GitHub push completed"
    }
    finally {
        Pop-Location
    }
}

$ResolvedRepoRoot = GetGitRepoRoot $RepoRoot
$ShouldPushAfterDeploy = TestHasPendingGitChanges $ResolvedRepoRoot
if ($ShouldPushAfterDeploy) {
    Write-Host "  git changes detected before version bump; deploy will push after verification"
} else {
    Write-Host "  no pending git changes before version bump; deploy will skip git push"
}

function SetWatchdog([bool]$on) {
    $val = if ($on) { "true" } else { "false" }
    $sh = "#!/bin/sh`nTOKEN=`$(cat /run/s6/container_environment/SUPERVISOR_TOKEN)`ncurl -s -X POST -H `"Authorization: Bearer `$TOKEN`" -H `"Content-Type: application/json`" -d '{`"watchdog`": $val}' http://supervisor/addons/$Slug/options"
    $bytes = [System.Text.Encoding]::ASCII.GetBytes($sh)
    [System.IO.File]::WriteAllBytes((Join-Path (Get-Location).Path "watchdog_tmp.sh"), $bytes)
    bash -c "bash hassh 'sh -s' < watchdog_tmp.sh"
    Remove-Item "watchdog_tmp.sh"
}

# 1. Bump version (always increments patch)
$sw = [System.Diagnostics.Stopwatch]::StartNew()
function Step([string]$s) { Write-Host "$s  +$([int]$sw.Elapsed.TotalSeconds)s" }
Step "[1/5] version"
$configText = Get-Content $ConfigPath -Raw
$current = [regex]::Match($configText, 'version: "([^"]+)"').Groups[1].Value
$p = $current.Split('.')
$p[2] = [string]([int]$p[2] + 1)
$NewVersion = $p -join '.'
Write-Host "  $current -> $NewVersion"
($configText -replace "version: `"$current`"", "version: `"$NewVersion`"") |
    Set-Content $ConfigPath -NoNewline

# 2. Test
Step "[2/5] test"
Push-Location $ResolvedRepoRoot
& "C:\Program Files\Go\bin\go.exe" test ./...
$testExit = $LASTEXITCODE
Pop-Location
if ($testExit -ne 0) { throw "Tests failed" }

# 3. Build linux-arm64 static binary
Step "[3/5] build"
$env:GOOS = "linux"
$env:GOARCH = "arm64"
$env:CGO_ENABLED = "0"
Push-Location $ResolvedRepoRoot
& "C:\Program Files\Go\bin\go.exe" build -ldflags "-X main.version=$NewVersion" -o $BinaryPath ./...
$buildExit = $LASTEXITCODE
Pop-Location
$env:GOOS = $null
$env:GOARCH = $null
$env:CGO_ENABLED = $null
if ($buildExit -ne 0) { throw "Build failed" }

# 4. Build Docker image on PC and push to ghcr.io
# The supervisor will pull this image directly -- no build on the Pi.
# The package must be public on ghcr.io so the Pi can pull without credentials.
Step "[4/5] push"
$dockerReady = $false
for ($d = 0; $d -lt 24; $d++) {
    & { $ErrorActionPreference = "SilentlyContinue"; docker info 2>&1 | Out-Null }
    if ($LASTEXITCODE -eq 0) { $dockerReady = $true; break }
    if ($d -eq 0) {
        Write-Host "  Docker not running - starting Docker Desktop..."
        Start-Process "C:\Program Files\Docker\Docker\Docker Desktop.exe"
    }
    $secs = $d * 5
    Write-Host "  waiting for Docker... ${secs}s"
    Start-Sleep 5
}
if (-not $dockerReady) { throw "Docker Desktop did not start in time" }
docker buildx build --platform linux/arm64 --push -t "${Image}:${NewVersion}" $ResolvedRepoRoot
if ($LASTEXITCODE -ne 0) { throw "Docker push failed" }
Write-Host "  pushed ${Image}:${NewVersion}"

# 5. Deploy: upload config.yaml to Pi, reload store, update add-on
# Only config.yaml is needed -- no Dockerfile or binary required on the Pi
# when image: is set in config.yaml.
Step "[5/5] deploy"
SetWatchdog $false
Write-Host "  watchdog disabled"

bash -c "bash hassh 'sudo mkdir -p $Remote'"
if ($LASTEXITCODE -ne 0) { throw "mkdir failed" }
$configLocal = ($ConfigPath -replace '\\','/')
bash -c "bash hassh 'sudo tee $Remote/config.yaml > /dev/null' < '$configLocal'"
if ($LASTEXITCODE -ne 0) { throw "config upload failed" }

$reloadOk = $false
for ($r = 0; $r -lt 3; $r++) {
    bash -c "bash hassh 'SUPERVISOR_TOKEN=`$(cat /run/s6/container_environment/SUPERVISOR_TOKEN) ha store reload'"
    if ($LASTEXITCODE -eq 0) { $reloadOk = $true; break }
    Write-Host "  store reload attempt $($r+1) failed, retrying..."
    Start-Sleep 5
}
if (-not $reloadOk) { Write-Host "  store reload failed after 3 attempts - continuing" }

$updateOk = $false
for ($r = 0; $r -lt 3; $r++) {
    Start-Sleep 3
    bash -c "bash hassh 'SUPERVISOR_TOKEN=`$(cat /run/s6/container_environment/SUPERVISOR_TOKEN) ha apps update $Slug 2>&1'"
    if ($LASTEXITCODE -eq 0) { $updateOk = $true; break }
    Write-Host "  update attempt $($r+1) failed, retrying..."
    Start-Sleep 10
}
if (-not $updateOk) {
    SetWatchdog $true
    throw "ha apps update failed after 3 attempts"
}
Write-Host "  update triggered"

$started = $false
for ($i = 0; $i -lt 30; $i++) {
    Start-Sleep 5
    $info = bash -c "bash hassh 'SUPERVISOR_TOKEN=`$(cat /run/s6/container_environment/SUPERVISOR_TOKEN) ha apps info $Slug 2>/dev/null | grep state'"
    Write-Host "  $info"
    if ($info -match "state: started") { $started = $true; break }
}
if (-not $started) {
    SetWatchdog $true
    throw "Add-on did not reach started state"
}
SetWatchdog $true
Write-Host "  watchdog re-enabled"

$here = (Get-Location).Path
$verifySh = "#!/bin/sh`nTOKEN=`$(cat /run/s6/container_environment/SUPERVISOR_TOKEN)`ncurl -s -H `"Authorization: Bearer `$TOKEN`" http://supervisor/addons/$Slug/logs | grep eNet"
$bytes = [System.Text.Encoding]::ASCII.GetBytes($verifySh)
[System.IO.File]::WriteAllBytes((Join-Path $here "verify_tmp.sh"), $bytes)
$startupLog = bash -c "bash hassh 'sh -s' < verify_tmp.sh"
Remove-Item "verify_tmp.sh"
Write-Host "  startup: $startupLog"

if ($ShouldPushAfterDeploy) {
    PushToGitHub $ResolvedRepoRoot $CommitMessage
} else {
    Write-Host "  skipping git push because the version bump was the only change"
}

if ($startupLog -match [regex]::Escape($NewVersion)) {
    Write-Host "OK: v$NewVersion confirmed in startup log"
} else {
    Write-Host "WARNING: v$NewVersion not in startup log - check manually"
}