<#
.SYNOPSIS
Installs the plugin DLLs that `just vm deploy` copied into plugin\ beside this
script, into the game's BepInEx\plugins\renki. The game must be closed,
because Windows locks a loaded DLL.
#>
. (Join-Path $PSScriptRoot 'common.ps1')
$s = Read-BotSettings

if (Get-Valheim) { throw 'Valheim is running; stop the bot first (RenkiStop).' }
$game = Split-Path $s.ValheimExe
$plugins = Join-Path $game 'BepInEx\plugins'
if (-not (Test-Path $plugins)) {
    throw "No $plugins. Install BepInExPack_Valheim into the game folder first."
}
$dest = Join-Path $plugins 'renki'
New-Item -ItemType Directory -Force -Path $dest | Out-Null
Copy-Item (Join-Path $PSScriptRoot 'plugin\*.dll') $dest -Force
Get-ChildItem $dest -Filter *.dll | ForEach-Object { Write-Host "installed $($_.FullName) ($($_.LastWriteTime))" }
