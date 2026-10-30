# Return one pinned image to the local capture recipe; tokens remain on the VM.
$ErrorActionPreference = 'Stop'
$cfg = Get-Content (Join-Path $PSScriptRoot 'controller.local.json') -Raw | ConvertFrom-Json
$token = [Environment]::GetEnvironmentVariable('RENKI_PANEL_TOKEN', 'User')
if (-not $token) { $token = $cfg.panel.token }
$base = 'http://' + $cfg.panel.listen
$headers = @{ Authorization = 'Bearer ' + $token }
for ($attempt = 0; $attempt -lt 3; $attempt++) {
    $metadata = Invoke-RestMethod -Uri ($base + '/api/camera') -Headers $headers -TimeoutSec 20
    if (-not $metadata.available) { throw 'No current camera image. Enable capture and join the world first.' }
    $temp = [IO.Path]::GetTempFileName()
    try {
        Invoke-WebRequest -UseBasicParsing -Uri ($base + $metadata.imageUrl) -Headers $headers -OutFile $temp -TimeoutSec 20
        $metadata | Add-Member -NotePropertyName jpegBase64 -NotePropertyValue ([Convert]::ToBase64String([IO.File]::ReadAllBytes($temp)))
        $metadata | ConvertTo-Json -Depth 12 -Compress
        exit 0
    } catch {
        if ($_.Exception.Response.StatusCode.value__ -ne 409 -or $attempt -eq 2) { throw }
    } finally { Remove-Item $temp -ErrorAction SilentlyContinue }
}
