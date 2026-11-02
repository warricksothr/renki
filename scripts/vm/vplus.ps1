# Install only the verified mod DLL; retain the existing game and BepInEx runtime.
param(
    [ValidateSet('Inspect', 'Install')][string]$Action = 'Inspect',
    [ValidatePattern('^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$')][string]$Version = '10.4.0.0',
    [ValidatePattern('^[0-9a-fA-F]{64}$')][string]$Sha256
)
. (Join-Path $PSScriptRoot 'common.ps1')
$s = Read-BotSettings
$game = Split-Path $s.ValheimExe
$plugins = Join-Path $game 'BepInEx\plugins'
if ($Action -eq 'Install') {
    if ((Get-Valheim) -or (Get-Controller $s)) { throw 'Stop Valheim and controller before installing.' }
    if (-not $Sha256) { throw 'Expected SHA256 is required.' }
    $backup = Get-ChildItem (Join-Path $PSScriptRoot 'backups') -Directory -Filter 'vplus-*' |
        Where-Object { Test-Path (Join-Path $_.FullName 'complete.json') } |
        Sort-Object Name -Descending | Select-Object -First 1
    if (-not $backup) { throw 'No completed migration backup; run just vm migration-backup.' }
    $complete = Get-Content (Join-Path $backup.FullName 'complete.json') -Raw | ConvertFrom-Json
    if (-not $complete.verified) { throw 'Backup is not verified.' }
    $manifest = Get-Content (Join-Path $backup.FullName 'manifest.json') -Raw | ConvertFrom-Json
    if (-not $manifest -or @($manifest).Count -lt 1) { throw 'Empty backup manifest.' }
    foreach ($entry in $manifest) {
        if ((Get-FileHash -LiteralPath (Join-Path $backup.FullName $entry.backup) -Algorithm SHA256).Hash -ne $entry.sha256) {
            throw 'Migration backup checksum mismatch.'
        }
    }
    $source = Join-Path $PSScriptRoot "staging\vplus-$Version\ValheimPlus.dll"
    if ((Get-FileHash -LiteralPath $source -Algorithm SHA256).Hash -ne $Sha256) { throw 'Staged DLL checksum mismatch.' }
    if ([Reflection.AssemblyName]::GetAssemblyName($source).Version.ToString() -ne $Version) { throw 'Staged assembly version mismatch.' }
    $existing = @(Get-ChildItem -LiteralPath $plugins -Filter '*.dll' -Recurse | Where-Object {
        try { [Reflection.AssemblyName]::GetAssemblyName($_.FullName).Name -eq 'ValheimPlus' } catch { $false }
    })
    if ($existing.Count) { throw 'V+ is already installed; reconcile existing copies before replacing anything.' }
    $cfg = Join-Path $game 'BepInEx\config\org.bepinex.plugins.valheim_plus.cfg'
    if (Test-Path -LiteralPath $cfg) { throw 'Existing V+ configuration; preserve and reconcile it before a default installation.' }
    if (Test-Path -LiteralPath (Join-Path $game 'BepInEx\config\valheim_plus.cfg')) {
        throw 'Legacy V+ configuration would override defaults; reconcile it before installation.'
    }
    $renki = @(Get-ChildItem -LiteralPath $plugins -Recurse -Filter 'Renki.*.dll' | ForEach-Object {
        @{ path = $_.FullName; sha256 = (Get-FileHash -LiteralPath $_.FullName -Algorithm SHA256).Hash }
    })
    if ($renki.Count -lt 2) { throw 'Expected the installed Renki plugin and core assemblies.' }
    $dest = Join-Path $plugins 'ValheimPlus'
    New-Item -ItemType Directory -Force -Path $dest | Out-Null
    $installed = Join-Path $dest 'ValheimPlus.dll'
    Copy-Item -LiteralPath $source -Destination $installed
    if ((Get-FileHash -LiteralPath $installed -Algorithm SHA256).Hash -ne $Sha256) { throw 'Installed checksum mismatch.' }
    foreach ($entry in $renki) {
        if ((Get-FileHash -LiteralPath $entry.path -Algorithm SHA256).Hash -ne $entry.sha256) { throw 'Renki assembly changed.' }
    }
    @{ installed = $installed; version = $Version; sha256 = $Sha256; backup = $backup.FullName; renkiUnchanged = $true } |
        ConvertTo-Json -Depth 3
    return
}
$mods = @(Get-ChildItem -LiteralPath $plugins -Recurse -Filter '*.dll' | ForEach-Object {
    $name = try { [Reflection.AssemblyName]::GetAssemblyName($_.FullName) } catch { $null }
    @{ path = $_.FullName.Substring($game.Length + 1); assembly = $name.Name; version = [string]$name.Version;
       sha256 = (Get-FileHash -LiteralPath $_.FullName -Algorithm SHA256).Hash }
})
$log = Join-Path $game 'BepInEx\LogOutput.log'
$lines = @()
if (Test-Path -LiteralPath $log) {
    # Explicit version/patch diagnostics only, never raw chat, endpoints or configs.
    $lines = @(Get-Content -LiteralPath $log | Where-Object {
        $_ -match 'BepInExPack Valheim version|BepInEx [0-9]|Valheim [Vv]ersion:|Network version check|Loading \[|ValheimPlus \[|[Rr]eceived config|[Rr]egistered [0-9]+ settings|[0-9]+ settings (changed|different)|re-applying patches|\[Error\s*:|\[Warning\s*:Valheim Plus\]'
    } | Select-Object -Last 45 | ForEach-Object { [string]$_ })
}
$settings = @{}
$cfg = Join-Path $game 'BepInEx\config\org.bepinex.plugins.valheim_plus.cfg'
if (Test-Path -LiteralPath $cfg) {
    $section = ''
    foreach ($line in Get-Content -LiteralPath $cfg) {
        if ($line -match '^\[([^\]]+)\]') { $section = $Matches[1] }
        elseif ($line -match '^(enabled|enforceMod|serverSyncsConfig|autoFuel|autoDeposit)\s*=\s*(true|false)\s*$') {
            $settings["$section.$($Matches[1])"] = $Matches[2] -eq 'true'
        }
    }
}
@{ gameRunning = [bool](Get-Valheim); controllerRunning = [bool](Get-Controller $s); mods = $mods;
   logUpdated = $(if (Test-Path $log) { (Get-Item $log).LastWriteTimeUtc.ToString('o') });
   diagnostics = $lines; localConfigFlags = $settings } | ConvertTo-Json -Depth 5
