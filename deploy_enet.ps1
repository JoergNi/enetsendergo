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
# release snapshot they are creating, e.g. .\deploy_enet.ps1 -CommitMessage "Fix scheduler drift".

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
$SshKey = "$env:USERPROFILE\.ssh\homevibeportal"
$SshHost = "root@192.168.178.47"
$SshPort = "22222"

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

function TestGitHasUpstream([string]$RepoPath) {
    & git -C $RepoPath rev-parse --abbrev-ref --symbolic-full-name "@{upstream}" 1>$null 2>$null
    return ($LASTEXITCODE -eq 0)
}

function CommitReleaseSnapshot([string]$RepoPath, [string]$CommitMessage) {
    $resolvedRepoRoot = GetGitRepoRoot $RepoPath

    Push-Location $resolvedRepoRoot
    try {
        & git add -A
        if ($LASTEXITCODE -ne 0) { throw "git add failed" }

        & git diff --cached --quiet
        if ($LASTEXITCODE -gt 1) { throw "git diff --cached failed" }
        if ($LASTEXITCODE -eq 0) {
            throw "no git changes to commit after version bump"
        }

        & git commit -m $CommitMessage | Out-Null
        if ($LASTEXITCODE -ne 0) { throw "git commit failed" }

        $revLines = & git rev-parse HEAD
        if ($LASTEXITCODE -ne 0) { throw "git rev-parse failed" }
        $releaseCommit = ($revLines | Select-Object -First 1).Trim()
        return $releaseCommit
    }
    finally {
        Pop-Location
    }
}

$ResolvedRepoRoot = GetGitRepoRoot $RepoRoot

function InvokeHaSsh([string]$Command) {
    & ssh.exe -i $SshKey -p $SshPort -o StrictHostKeyChecking=no -o ConnectTimeout=10 `
        -o MACs=hmac-sha2-256-etm@openssh.com $SshHost $Command
}

function InvokeHaScript([string]$Script) {
    $encoded = [Convert]::ToBase64String([System.Text.Encoding]::UTF8.GetBytes($Script))
    InvokeHaSsh "printf '%s' '$encoded' | base64 -d | sh"
}

function InvokeHaSupervisor([string]$Command) {
    InvokeHaScript @"
export SUPERVISOR_TOKEN=`$(cat /run/s6/container_environment/SUPERVISOR_TOKEN)
$Command
"@
}

function CopyToHa([string]$LocalPath, [string]$RemotePath) {
    & scp.exe -i $SshKey -P $SshPort -o StrictHostKeyChecking=no -o ConnectTimeout=10 `
        -o MACs=hmac-sha2-256-etm@openssh.com $LocalPath "${SshHost}:${RemotePath}"
}

function AssertAddonIsNotDetached() {
    $appInfoLines = @(InvokeHaSupervisor "ha apps info $Slug")
    if ($LASTEXITCODE -ne 0) { throw "ha apps info failed for $Slug" }
    $appInfo = $appInfoLines -join "`n"
    $detachedMatch = [regex]::Match($appInfo, '(?m)^detached:\s*(.+)$')
    if (-not $detachedMatch.Success) {
        throw "Could not determine detached state for $Slug"
    }

    $detached = $detachedMatch.Groups[1].Value.Trim()
    if ($detached -eq "true") {
        throw "Add-on $Slug is detached in Home Assistant. Reattach or reinstall it from the local repository before using deploy_enet.ps1. Detached add-ons do not pick up config.yaml version changes via store reload/apps update."
    }
}

function SetWatchdog([bool]$on) {
    $val = if ($on) { "true" } else { "false" }
    $script = @'
TOKEN=$(cat /run/s6/container_environment/SUPERVISOR_TOKEN); curl -s -X POST -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d '{{"watchdog": {0}}}' http://supervisor/addons/{1}/options
'@ -f $val, $Slug
    InvokeHaScript $script | Out-Null
    if ($LASTEXITCODE -ne 0) { throw "watchdog update failed" }
}

# 0. Preflight: refuse to deploy against a detached add-on
$sw = [System.Diagnostics.Stopwatch]::StartNew()
function Step([string]$s) { Write-Host "$s  +$([int]$sw.Elapsed.TotalSeconds)s" }
Step "[0/6] preflight"
AssertAddonIsNotDetached

# 1. Unit tests
Step "[1/6] test"
Push-Location $ResolvedRepoRoot
& "C:\Program Files\Go\bin\go.exe" test ./...
$testExit = $LASTEXITCODE
Pop-Location
if ($testExit -ne 0) { throw "Tests failed" }

# 2. Bump version
Step "[2/6] version"
$configText = Get-Content $ConfigPath -Raw
$current = [regex]::Match($configText, 'version: "([^"]+)"').Groups[1].Value
$p = $current.Split('.')
$p[2] = [string]([int]$p[2] + 1)
$NewVersion = $p -join '.'
Write-Host "  $current -> $NewVersion"
($configText -replace "version: `"$current`"", "version: `"$NewVersion`"") |
    Set-Content $ConfigPath -NoNewline

# 3. Commit the release snapshot
Step "[3/6] commit"
$GitCommit = CommitReleaseSnapshot $ResolvedRepoRoot $CommitMessage
Write-Host "  committed release snapshot: $GitCommit"

# 4. Build linux-arm64 static binary
Step "[4/6] build"
$env:GOOS = "linux"
$env:GOARCH = "arm64"
$env:CGO_ENABLED = "0"
Push-Location $ResolvedRepoRoot
& "C:\Program Files\Go\bin\go.exe" build -ldflags "-X main.version=$NewVersion -X main.gitCommit=$GitCommit" -o $BinaryPath ./...
$buildExit = $LASTEXITCODE
Pop-Location
$env:GOOS = $null
$env:GOARCH = $null
$env:CGO_ENABLED = $null
if ($buildExit -ne 0) { throw "Build failed" }

# 5. Build Docker image on PC and push to ghcr.io
# The supervisor will pull this image directly -- no build on the Pi.
# The package must be public on ghcr.io so the Pi can pull without credentials.
Step "[5/6] push"
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

# 6. Deploy: upload config.yaml to Pi, reload store, update add-on
# Only config.yaml is needed -- no Dockerfile or binary required on the Pi
# when image: is set in config.yaml.
Step "[6/6] deploy"
SetWatchdog $false
Write-Host "  watchdog disabled"

InvokeHaSsh "sudo mkdir -p $Remote"
if ($LASTEXITCODE -ne 0) { throw "mkdir failed" }
CopyToHa $ConfigPath "/tmp/enetsender_config.yaml"
if ($LASTEXITCODE -ne 0) { throw "config upload failed" }
InvokeHaScript @"
sudo mv /tmp/enetsender_config.yaml $Remote/config.yaml
"@
if ($LASTEXITCODE -ne 0) { throw "config upload failed" }

$reloadOk = $false
for ($r = 0; $r -lt 3; $r++) {
    InvokeHaSupervisor "ha store reload"
    if ($LASTEXITCODE -eq 0) { $reloadOk = $true; break }
    Write-Host "  store reload attempt $($r+1) failed, retrying..."
    Start-Sleep 5
}
if (-not $reloadOk) { Write-Host "  store reload failed after 3 attempts - continuing" }

$updateOk = $false
for ($r = 0; $r -lt 3; $r++) {
    Start-Sleep 3
    InvokeHaSupervisor "ha apps update $Slug 2>&1"
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
    $info = InvokeHaSupervisor "ha apps info $Slug 2>/dev/null | grep state"
    Write-Host "  $info"
    if ($info -match "state: started") { $started = $true; break }
}
if (-not $started) {
    SetWatchdog $true
    throw "Add-on did not reach started state"
}
SetWatchdog $true
Write-Host "  watchdog re-enabled"

$appInfoLines = @(InvokeHaSupervisor "ha apps info $Slug")
$appInfo = $appInfoLines -join "`n"
$versionMatch = [regex]::Match($appInfo, '(?m)^version:\s*(.+)$')
$stateMatch = [regex]::Match($appInfo, '(?m)^state:\s*(.+)$')
$deployedVersion = if ($versionMatch.Success) { $versionMatch.Groups[1].Value.Trim() } else { "unknown" }
$deployedState = if ($stateMatch.Success) { $stateMatch.Groups[1].Value.Trim() } else { "unknown" }
Write-Host "  deployed version: $deployedVersion"
Write-Host "  deployed state: $deployedState"

$recentLogs = InvokeHaSupervisor "ha apps logs $Slug | tail -n 10"
Write-Host "  recent logs:`n$recentLogs"

if ($deployedVersion -eq $NewVersion -and $deployedState -eq "started") {
    Write-Host "  verified: $Slug v$deployedVersion is started"
    if (TestGitHasUpstream $ResolvedRepoRoot) {
        Push-Location $ResolvedRepoRoot
        try {
            & git push
            if ($LASTEXITCODE -ne 0) { throw "git push failed" }
        }
        finally {
            Pop-Location
        }
        Write-Host "  GitHub push completed"
    } else {
        Write-Host "  skipping git push because no upstream is configured"
    }
    Write-Host "OK: v$NewVersion deployed and started"
} else {
    throw "Post-deploy verification failed: expected v$NewVersion started, got version=$deployedVersion state=$deployedState"
}
