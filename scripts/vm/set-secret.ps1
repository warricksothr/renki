<#
.SYNOPSIS
Stores one secret for the controller, read from standard input.

.DESCRIPTION
`just vm secret NAME` runs this over SSH and pipes the value in, so it is
never on a command line, in a log, or on screen. It goes into
secrets.local.json beside this script, which only this account and SYSTEM
may read. (Not Administrators as a group: the RenkiStart task runs with a
limited token, in which that group only denies.) start-bot.ps1 puts every entry into the controller's
environment, where the configuration names it, such as llm.terva.apiKeyEnv.

.PARAMETER Name
The environment variable, such as CPA_API_KEY.

.PARAMETER Remove
Delete the entry instead.
#>
param(
    [Parameter(Mandatory = $true)][string]$Name,
    [switch]$Remove
)

$ErrorActionPreference = 'Stop'
if ($Name -notmatch '^[A-Z_][A-Z0-9_]*$') { throw "Name must look like CPA_API_KEY, not $Name" }
. (Join-Path $PSScriptRoot 'common.ps1')

$secrets = Read-Secrets
if ($Remove) {
    $secrets.Remove($Name) | Out-Null
    Write-Host "removed $Name"
} else {
    $value = [Console]::In.ReadToEnd().Trim()
    if (-not $value) { throw 'No value on standard input; nothing changed.' }
    $secrets[$Name] = $value
    Write-Host "stored $Name ($($value.Length) characters)"
}

$file = Join-Path $PSScriptRoot 'secrets.local.json'
# Lock the file down before the value goes in, so it never sits under the
# folder's ACL: only this account (the one RenkiStart runs as) and SYSTEM,
# and nothing inherited. By SID: over SSH $env:USERDOMAIN is WORKGROUP,
# not the machine, so a name would be wrong.
if (-not (Test-Path $file)) { New-Item -ItemType File -Path $file | Out-Null }
$me = [Security.Principal.WindowsIdentity]::GetCurrent().User.Value
icacls.exe $file /inheritance:r /grant:r "*${me}:F" '*S-1-5-18:F' | Out-Null
if ($LASTEXITCODE -ne 0) { throw "could not restrict $file" }
icacls.exe $file /remove:g '*S-1-5-32-544' | Out-Null
if ($LASTEXITCODE -ne 0) { throw "could not restrict $file" }
$secrets | ConvertTo-Json | Set-Content $file -Encoding ascii
Write-Host 'restart the controller to use it: just vm restart'
