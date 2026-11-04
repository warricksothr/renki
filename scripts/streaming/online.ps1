$ErrorActionPreference = 'Stop'
$root = $PSScriptRoot
$adapter = Join-Path $root 'obs-adapter.exe'
$config = Join-Path $root 'private/adapter.json'
$obsExe = (Join-Path $root 'obs/bin/64bit/obs64.exe').Replace('/','\')

# Restore the installed stack; never install tools, rotate credentials or alter scenes.
foreach ($path in @($adapter, $config, $obsExe, (Join-Path $root 'private/relay.json'))) {
    if (-not (Test-Path $path)) { throw 'Private video setup is incomplete; see docs/WINDOWS-VIDEO.md' }
}
$existing = @(Get-Process obs64 -ErrorAction SilentlyContinue)
if ($existing.Count -gt 1) { throw 'Multiple OBS instances are running' }
if (@($existing | Where-Object { $_.Path -ne $obsExe }).Count) {
    throw 'A different OBS installation is already running'
}
# OBS 33 removed --disable-shutdown-check. Its stale run_* sentinels block
# startup in a modal dialog. Archive them only with this dedicated OBS stopped.
if ($existing.Count -eq 1) {
    $log = Get-ChildItem (Join-Path $root 'obs/config/obs-studio/logs') -Filter '*.txt' | Sort-Object LastWriteTime -Descending | Select-Object -First 1
    $startupOnly = $false
    if ($log -and [Math]::Abs(($log.CreationTimeUtc - $existing[0].StartTime.ToUniversalTime()).TotalSeconds) -lt 10) {
        $lines = @((((Get-Content $log.FullName -Raw) -replace [string][char]0,'') -split "`n") | Where-Object { $_.Trim() })
        $startupOnly = $lines.Count -eq 1 -and $lines[0].Trim() -match '^\d\d:\d\d:\d\d\.\d+: Crash or unclean shutdown detected$'
    }
    if ($startupOnly -and -not (Get-NetTCPConnection -LocalPort 4455 -State Listen -ErrorAction SilentlyContinue)) {
        Stop-ScheduledTask -TaskName RenkiOBS
        if (-not $existing[0].WaitForExit(5000)) { Stop-Process -Id $existing[0].Id -Force; $existing[0].WaitForExit(5000) | Out-Null }
        Write-Host 'Recovered dedicated OBS blocked at its unclean-shutdown dialog'
    }
}
if (-not (Get-Process obs64 -ErrorAction SilentlyContinue)) {
    $sentinels = @(Get-ChildItem (Join-Path $root 'obs/config/obs-studio/.sentinel') -Filter 'run_*' -File -ErrorAction SilentlyContinue)
    if ($sentinels.Count) {
        $archive = Join-Path $root ('private/startup-recovery/' + (Get-Date -Format 'yyyyMMddTHHmmssfff'))
        New-Item -ItemType Directory -Path $archive -Force | Out-Null
        $sentinels | Move-Item -Destination $archive
        Write-Host ('Archived ' + $sentinels.Count + ' stale dedicated OBS run markers; crash logs retained')
    }
}
$names = @('RenkiOBS','RenkiRelay','RenkiCapture','RenkiCaptureWatchdog')
# Resolve every task before starting any, so missing setup fails promptly.
$tasks = @($names | ForEach-Object { Get-ScheduledTask -TaskName $_ -ErrorAction Stop })
foreach ($task in $tasks) {
    if ($task.State -ne 'Running') { Start-ScheduledTask -InputObject $task }
}
function Read-OBSStatus {
    # Windows PowerShell turns native stderr into an error record even when redirected.
    $ErrorActionPreference = 'Continue'
    $raw = & $adapter -config $config -action status 2>$null
    if ($LASTEXITCODE -ne 0) { return $null }
    try { return (($raw -join "`n") | ConvertFrom-Json -ErrorAction Stop) } catch { return $null }
}
$deadline = (Get-Date).AddSeconds(60)
do {
    $status = Read-OBSStatus
    if ($null -ne $status) { break }
    if ((Get-Date) -ge $deadline) { throw 'Authenticated OBS did not become ready within 60 seconds' }
    Start-Sleep -Seconds 2
} while ($true)

# local-start verifies dedicated scenes and only permits the fixed loopback output.
& $adapter -config $config -action local-start
if ($LASTEXITCODE -ne 0) { throw 'Private OBS output could not start' }
$deadline = (Get-Date).AddSeconds(30)
do {
    $status = Read-OBSStatus
    if ($null -ne $status -and $status.stream.outputActive) { break }
    if ((Get-Date) -ge $deadline) { throw 'Private OBS output did not become active within 30 seconds' }
    Start-Sleep -Seconds 2
} while ($true)
foreach ($name in $names) {
    if ((Get-ScheduledTask -TaskName $name).State -ne 'Running') { throw "Private video task failed: $name" }
}
Write-Host 'Private OBS output active; capture verifier and watchdog running'
$status | ConvertTo-Json -Depth 6 -Compress
