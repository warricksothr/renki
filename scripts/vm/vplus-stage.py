#!/usr/bin/env python3
"""Stage a pinned official V+ DLL and verify GitHub's release asset digest."""
import hashlib
import json
from pathlib import Path
import re
import sys
import urllib.request

ROOT = Path(__file__).resolve().parents[2]


def fetch(url):
    request = urllib.request.Request(url, headers={'User-Agent': 'Renki-VPlus-migration'})
    with urllib.request.urlopen(request, timeout=60) as response:
        return response.read()


def main():
    version = sys.argv[1]
    if not re.fullmatch(r'[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+', version):
        raise ValueError('Use an exact four-part release version')
    release = json.loads(fetch('https://api.github.com/repos/Grantapher/ValheimPlus/releases/tags/' + version))
    if release['tag_name'] != version or release['draft']:
        raise ValueError('Not the requested published release')
    asset, = [item for item in release['assets'] if item['name'] == 'ValheimPlus.dll']
    digest = asset.get('digest', '')
    if not re.fullmatch(r'sha256:[0-9a-f]{64}', digest):
        raise ValueError('Release asset has no SHA256 digest')
    data = fetch(asset['browser_download_url'])
    sha = hashlib.sha256(data).hexdigest()
    if digest != 'sha256:' + sha or len(data) != asset['size']:
        raise ValueError('Downloaded DLL does not match the official asset')
    folder = ROOT / '.checks' / 'valheim-plus' / version
    folder.mkdir(parents=True, exist_ok=True)
    (folder / 'ValheimPlus.dll').write_bytes(data)
    manifest = dict(version=version, sha256=sha, bytes=len(data), url=asset['browser_download_url'],
                    release=release['html_url'], published=release['published_at'])
    (folder / 'release.json').write_text(json.dumps(manifest, indent=2) + '\n')
    print(json.dumps(dict(manifest, staged=str(folder))))


if __name__ == '__main__':
    main()
