# Read-only deployment diagnosis; never starts the game or changes its files.
. (Join-Path $PSScriptRoot 'common.ps1')
$s = Read-BotSettings
$plugins = Join-Path (Split-Path $s.ValheimExe) 'BepInEx\plugins'
$libraries = @(Get-ChildItem $plugins -Recurse -File | Where-Object { $_.Name -in @('Renki.Core.dll','Renki.Plugin.dll') } | ForEach-Object {
    $moduleId = try { [Reflection.Assembly]::ReflectionOnlyLoadFrom($_.FullName).ManifestModule.ModuleVersionId.ToString() } catch { 'unavailable: ' + $_.Exception.Message }
    @{ path = $_.FullName; sha256 = (Get-FileHash $_.FullName -Algorithm SHA256).Hash; mvid = $moduleId }
})
@{ libraries=$libraries } | ConvertTo-Json -Depth 5
