<#
.SYNOPSIS
Installs the plugin DLLs that `just vm deploy` copied into plugin\ beside this
script, into the game's BepInEx\plugins\renki. The game must be closed,
because Windows locks a loaded DLL.
#>
. (Join-Path $PSScriptRoot 'common.ps1')
$s = Read-BotSettings

if (Get-Valheim) { throw 'Valheim is running; stop the bot first (RenkiStop).' }
$manifest = Join-Path $PSScriptRoot 'SHA256SUMS'
if (-not (Test-Path $manifest)) { throw 'No package checksum manifest; rebuild with just package.' }
foreach ($line in Get-Content $manifest) {
    if ($line -notmatch '^([0-9a-fA-F]{64}) [ *](.+)$') { throw 'Malformed package checksum manifest.' }
    $expected = $Matches[1]
    $file = Join-Path $PSScriptRoot $Matches[2]
    if ((Get-FileHash -Algorithm SHA256 $file).Hash -ne $expected) { throw "Package checksum mismatch: $file" }
}
$game = Split-Path $s.ValheimExe
$plugins = Join-Path $game 'BepInEx\plugins'
if (-not (Test-Path $plugins)) {
    throw "No $plugins. Install BepInExPack_Valheim into the game folder first."
}
$dest = Join-Path $plugins 'renki'
New-Item -ItemType Directory -Force -Path $dest | Out-Null
foreach ($source in Get-ChildItem (Join-Path $PSScriptRoot 'plugin') -Filter *.dll) {
    $installed = Join-Path $dest $source.Name
    Copy-Item $source.FullName $installed -Force
    if ((Get-FileHash -Algorithm SHA256 $installed).Hash -ne (Get-FileHash -Algorithm SHA256 $source.FullName).Hash) {
        throw "Installed plugin checksum mismatch: $installed"
    }
    Write-Host "verified installed $installed"
}
Write-Host 'controller and plugin package checksums verified'
