<#
.SYNOPSIS
Inspects migration backup sources; -Create copies and verifies them while offline.
The manifest contains restore paths. Backup contents stay on the VM and are
restricted to this account and SYSTEM because they include local secrets.
#>
param([switch]$Create)

. (Join-Path $PSScriptRoot 'common.ps1')
$s = Read-BotSettings
$game = Split-Path $s.ValheimExe
$sources = [Collections.Generic.List[object]]::new()

function Add-Source([string]$path, [string]$label) {
    if ($path -and (Test-Path -LiteralPath $path)) {
        $sources.Add([pscustomobject]@{ source = (Get-Item -LiteralPath $path).FullName; label = $label })
    }
}

Add-Source (Join-Path $game 'BepInEx') 'game/BepInEx'
foreach ($name in 'winhttp.dll', 'doorstop_config.ini', '.doorstop_version') {
    Add-Source (Join-Path $game $name) "game/$name"
}
Add-Source $s.ControllerExe 'renki/controller.exe'
Add-Source $s.ControllerConfig 'renki/controller-config.json'
foreach ($file in Get-ChildItem -LiteralPath $PSScriptRoot -File -Filter '*.local.*') {
    Add-Source $file.FullName ("renki/local/" + $file.Name)
}
$dataDir = 'data'
if ($s.ControllerConfig -and (Test-Path -LiteralPath $s.ControllerConfig)) {
    $config = Get-Content -LiteralPath $s.ControllerConfig -Raw | ConvertFrom-Json
    if ($config.dataDir) { $dataDir = [string]$config.dataDir }
}
if (-not [IO.Path]::IsPathRooted($dataDir)) { $dataDir = Join-Path $PSScriptRoot $dataDir }
Add-Source $dataDir 'renki/data'
$saves = Join-Path $env:USERPROFILE 'AppData\LocalLow\IronGate\Valheim'
foreach ($name in 'characters', 'characters_local', 'worlds', 'worlds_local') {
    Add-Source (Join-Path $saves $name) "saves/local/$name"
}
$steam = Get-ItemProperty -LiteralPath 'HKCU:\Software\Valve\Steam' -ErrorAction SilentlyContinue
if ($steam.SteamPath) {
    $userdata = Join-Path $steam.SteamPath 'userdata'
    if (Test-Path -LiteralPath $userdata) {
        foreach ($account in Get-ChildItem -LiteralPath $userdata -Directory) {
            Add-Source (Join-Path $account.FullName '892970') ("saves/steam/" + $account.Name + '/892970')
        }
    }
}

$files = [Collections.Generic.List[object]]::new()
foreach ($source in $sources) {
    $item = Get-Item -LiteralPath $source.source
    $entries = if ($item.PSIsContainer) { @(Get-ChildItem -LiteralPath $item.FullName -Recurse -File -Force) } else { @($item) }
    foreach ($file in $entries) {
        # Do not follow directory junctions into unrelated profiles or backup trees.
        $parent = $file
        while ($parent -and $parent.FullName.Length -ge $item.FullName.Length) {
            if ($parent.Attributes -band [IO.FileAttributes]::ReparsePoint) { throw "Reparse point in backup source: $($parent.FullName)" }
            $parent = if ($parent -is [IO.FileInfo]) { $parent.Directory } else { $parent.Parent }
        }
        $relative = if ($item.PSIsContainer) { $file.FullName.Substring($item.FullName.Length + 1) } else { '' }
        $destination = if ($relative) { Join-Path $source.label $relative } else { $source.label }
        $files.Add([pscustomobject]@{ source = $file.FullName; backup = $destination; bytes = $file.Length })
    }
}
$gameRunning = [bool](Get-Valheim)
$controllerRunning = [bool](Get-Controller $s)
$report = [ordered]@{
    at = [DateTime]::UtcNow.ToString('o')
    gameRunning = $gameRunning
    controllerRunning = $controllerRunning
    sources = @($sources.ToArray())
    fileCount = $files.Count
    bytes = ($files | Measure-Object -Property bytes -Sum).Sum
    backup = $null
    verified = $false
}
if ($Create) {
    if ($gameRunning -or $controllerRunning) { throw 'Stop Valheim and the local controller with just vm stop before backing up.' }
    if (-not (Test-Path -LiteralPath (Join-Path $game 'BepInEx'))) { throw 'No BepInEx installation to back up.' }
    $root = Join-Path $PSScriptRoot ('backups\vplus-' + [DateTime]::UtcNow.ToString('yyyyMMddTHHmmssfffZ'))
    New-Item -ItemType Directory -Path $root | Out-Null
    $me = [Security.Principal.WindowsIdentity]::GetCurrent().User.Value
    icacls.exe $root /inheritance:r /grant:r "*${me}:(OI)(CI)F" '*S-1-5-18:(OI)(CI)F' | Out-Null
    if ($LASTEXITCODE -ne 0) { throw 'Could not restrict the backup directory.' }
    icacls.exe $root /remove:g '*S-1-5-32-544' | Out-Null
    if ($LASTEXITCODE -ne 0) { throw 'Could not remove inherited administrator access.' }
    $manifest = [Collections.Generic.List[object]]::new()
    foreach ($file in $files) {
        $before = (Get-FileHash -LiteralPath $file.source -Algorithm SHA256).Hash
        $dest = Join-Path $root $file.backup
        New-Item -ItemType Directory -Force -Path (Split-Path $dest) | Out-Null
        Copy-Item -LiteralPath $file.source -Destination $dest
        $copied = (Get-FileHash -LiteralPath $dest -Algorithm SHA256).Hash
        $after = (Get-FileHash -LiteralPath $file.source -Algorithm SHA256).Hash
        if ($before -ne $copied -or $before -ne $after) { throw "Backup verification failed: $($file.source)" }
        $manifest.Add([pscustomobject]@{ source = $file.source; backup = $file.backup; sha256 = $copied; bytes = $file.bytes })
    }
    if ((Get-Valheim) -or (Get-Controller $s)) { throw 'Game or controller started during backup; snapshot is incomplete.' }
    $manifest.ToArray() | ConvertTo-Json -Depth 4 | Set-Content -LiteralPath (Join-Path $root 'manifest.json') -Encoding UTF8
    $report.backup = $root
    $report.verified = $true
    $report | ConvertTo-Json -Depth 5 | Set-Content -LiteralPath (Join-Path $root 'complete.json') -Encoding UTF8
}
$report | ConvertTo-Json -Depth 5
