# The VM side of the lease. scripts/vm/lease.py sends this with
# -EncodedCommand after a preamble that sets $Dir, $Action, $Expect and
# $Content, so no copy of it has to be on the VM first. Every decision is made
# in lease.py; this only reads the file and swaps it atomically.
# Windows PowerShell 5.1 or later.
#
#   Read   prints {"now", "sha256", "content"}: the VM's UTC clock, the
#          lease file's SHA-256 ('' when absent) and its bytes in base64.
#   Write  replaces the file with base64 $Content, or deletes it when $Content
#          is '', only if its SHA-256 still equals $Expect ('' = absent).
#          Prints ok, or conflict and exits 3 when someone changed it first.

$ErrorActionPreference = 'Stop'
$path = Join-Path $Dir 'lease.json'

# The file's bytes, or $null when absent.
function Get-LeaseBytes {
    if (-not (Test-Path -LiteralPath $path)) { return $null }
    return ,[IO.File]::ReadAllBytes($path)
}

# The SHA-256 of exactly the bytes given, so a hash always matches the content
# it is reported with; '' for an absent file.
function Get-LeaseHash($bytes) {
    if ($null -eq $bytes) { return '' }
    $sha = [Security.Cryptography.SHA256]::Create()
    try {
        return ([BitConverter]::ToString($sha.ComputeHash($bytes)) -replace '-', '').ToLowerInvariant()
    } finally { $sha.Dispose() }
}

if ($Action -eq 'Read') {
    $bytes = Get-LeaseBytes
    $content = ''
    if ($null -ne $bytes) { $content = [Convert]::ToBase64String($bytes) }
    [ordered]@{
        now     = [DateTime]::UtcNow.ToString('yyyy-MM-ddTHH:mm:ssZ')
        sha256  = Get-LeaseHash $bytes
        content = $content
    } | ConvertTo-Json -Compress
    exit 0
}

if ($Action -ne 'Write') { throw "unknown lease action '$Action'" }

New-Item -ItemType Directory -Force -Path $Dir | Out-Null
# One writer at a time: the lock file is opened without sharing, so a second
# writer waits here rather than interleaving its check and its swap.
$lock = $null
for ($i = 0; $i -lt 100 -and -not $lock; $i++) {
    try {
        $lock = [IO.File]::Open((Join-Path $Dir 'lease.lock'), 'OpenOrCreate', 'ReadWrite', 'None')
    } catch { Start-Sleep -Milliseconds 100 }
}
if (-not $lock) { 'the lease lock stayed busy for 10 s'; exit 4 }
try {
    if ((Get-LeaseHash (Get-LeaseBytes)) -ne $Expect) { 'conflict'; exit 3 }
    if ($Content -eq '') {
        Remove-Item -LiteralPath $path -Force -ErrorAction SilentlyContinue
    } else {
        $tmp = "$path.tmp"
        [IO.File]::WriteAllBytes($tmp, [Convert]::FromBase64String($Content))
        if (Test-Path -LiteralPath $path) {
            [IO.File]::Replace($tmp, $path, [NullString]::Value)
        } else {
            [IO.File]::Move($tmp, $path)
        }
    }
    'ok'
} finally { $lock.Dispose() }
