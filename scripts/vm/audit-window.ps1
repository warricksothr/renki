<# Stream only one time window of audit records, including state/perception. #>
param(
    [Parameter(Mandatory=$true)][DateTimeOffset]$Since,
    [Parameter(Mandatory=$true)][DateTimeOffset]$Until
)
$ErrorActionPreference = 'Stop'
if ($Until -le $Since) { throw 'Until must be after Since' }
Get-ChildItem (Join-Path $PSScriptRoot 'audit') -Filter 'audit-*.jsonl' | Sort-Object Name | ForEach-Object {
    $stream = [IO.File]::Open($_.FullName, [IO.FileMode]::Open, [IO.FileAccess]::Read, [IO.FileShare]::ReadWrite)
    $reader = [IO.StreamReader]::new($stream)
    try {
        while ($null -ne ($line = $reader.ReadLine())) {
            # Check the timestamp before decoding potentially large perception frames.
            if ($line -match '^\{"at":"([^"]+)"') {
                $at = [DateTimeOffset]::Parse($Matches[1])
                if ($at -ge $Since -and $at -lt $Until) { $line }
            }
        }
    } finally { $reader.Dispose() }
}
