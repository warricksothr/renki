#!/usr/bin/env python3
"""Stage pinned private Windows video tools; credentials never leave BLIGHT."""
import argparse
import hashlib
import os
from pathlib import Path
import subprocess
import urllib.request

ROOT = Path(__file__).resolve().parents[2]
OUT = ROOT / '.checks/streaming/windows'
ASSETS = [
    ('obs.zip', 'https://github.com/obsproject/obs-studio/releases/download/33.0.1/OBS-Studio-33.0.1-Windows-x64.zip', '0d9436451c2d772b33428e3fdae9776f071895c335de17283176306fb72651d3'),
    ('mediamtx.zip', 'https://github.com/bluenviron/mediamtx/releases/download/v1.21.3/mediamtx_v1.21.3_windows_amd64.zip', 'b2f5246b07cb72961df03b8d5e8cad43dc657bcf3f49a87bb031bdf22721af74'),
]

def digest_file(path):
    with path.open('rb') as stream:
        return hashlib.file_digest(stream, 'sha256').hexdigest()

def downloads():
    OUT.mkdir(parents=True, exist_ok=True)
    for name, url, digest in ASSETS:
        path = OUT / name
        if not path.exists() or digest_file(path) != digest:
            print('Downloading pinned ' + name, flush=True)
            with urllib.request.urlopen(url, timeout=60) as response, path.with_suffix('.tmp').open('wb') as stream:
                while data := response.read(1024 * 1024):
                    stream.write(data)
            path.with_suffix('.tmp').replace(path)
        if digest_file(path) != digest:
            path.unlink()
            raise SystemExit('Release checksum mismatch: ' + name)
        print('Verified ' + name, flush=True)

def remote(action):
    target = os.environ.get('RENKI_VM') or (ROOT / 'vm.local').read_text().strip()
    if not target or target.startswith('-'):
        raise SystemExit('Invalid VM target')
    base = os.environ.get('RENKI_VM_DIR', 'C:/renki')
    if base != 'C:/renki':
        raise SystemExit('Windows streaming currently requires C:/renki')
    ssh = ['ssh', '-o', 'BatchMode=yes', '-o', 'ConnectTimeout=10', target]
    scp = ['scp', '-q', '-o', 'BatchMode=yes']
    subprocess.run(ssh + ["New-Item -ItemType Directory -Force C:/renki/streaming | Out-Null"], check=True)
    files = [ROOT / 'scripts/streaming/windows.ps1']
    if action == 'slate':
        files += [ROOT / '.checks/streaming/windows/standby-obs.mp4']
    if action == 'setup':
        downloads()
        files += [OUT / name for name, _, _ in ASSETS]
        files += [ROOT / 'internal/panel/static/slate.html', ROOT / '.checks/streaming/windows/standby-obs.mp4']
    subprocess.run(scp + list(map(str, files)) + [target + ':/C:/renki/streaming/'], check=True)
    if action == 'adapter':
        binary = ROOT / 'dist/obs-adapter.exe'
        subprocess.run(scp + [str(binary), target + ':/C:/renki/streaming/obs-adapter.next.exe'], check=True)
        digest = digest_file(binary)
        subprocess.run(ssh + ["if ((Get-FileHash C:/renki/streaming/obs-adapter.next.exe -Algorithm SHA256).Hash.ToLowerInvariant() -ne '" + digest + "') { throw 'Adapter checksum mismatch' }"], check=True)
    subprocess.run(ssh + ['powershell -NoProfile -ExecutionPolicy Bypass -File C:/renki/streaming/windows.ps1 -Action ' + action], check=True)
    if action in ('snapshot', 'game-snapshot'):
        artifacts = ROOT / '.checks/obs-windows'
        artifacts.mkdir(parents=True, exist_ok=True)
        subprocess.run(scp + [target + ':/C:/renki/streaming/slate.html.snapshot.png', str(artifacts / (action + '.png'))], check=True)
        print('Saved OBS source diagnostic: ' + str(artifacts / (action + '.png')))

if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('action', choices=['downloads', 'setup', 'status', 'adapter', 'start', 'stop', 'relay', 'local-start', 'local-stop', 'snapshot', 'game-snapshot', 'enable', 'https', 'codec', 'beacon', 'slate', 'trial-source-loss', 'trial-watchdog'])
    args = parser.parse_args()
    if args.action == 'downloads':
        downloads()
    else:
        remote(args.action)
