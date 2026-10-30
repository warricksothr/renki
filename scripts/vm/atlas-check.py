#!/usr/bin/env python3
"""Read-only atlas capture/retention proof; full evidence stays in .checks/."""
import argparse
import datetime
import json
from pathlib import Path
import subprocess
import tempfile
import urllib.parse

ROOT = Path(__file__).resolve().parents[2]


def samples(tiles):
    out = {}
    for tile in tiles:
        for i, height in enumerate(tile['heights']):
            if height is None:
                continue
            key = f"{tile['x'] * 4 + i % 5},{tile['z'] * 4 + i // 5}"
            value = [height, tile['biomes'][i], tile['observedAt'][i], tile['sources'][i]]
            if key in out and out[key] != value:
                raise RuntimeError(f"Conflicting overlapping atlas border at {key}")
            out[key] = value
    return out


def retained(previous, current):
    missing = set(previous) - set(current)
    if missing:
        raise RuntimeError(f'{len(missing)} baseline samples lost')
    def instant(value):
        return datetime.datetime.fromisoformat(value.replace('Z', '+00:00'))
    older = sum(instant(current[k][2]) < instant(v[2]) for k, v in previous.items())
    if older:
        raise RuntimeError(f'{older} samples regressed in measurement time')
    return sum(current[k][:2] != v[:2] for k, v in previous.items())


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--baseline', type=Path, help='compare the same saved world/viewport to a retained capture')
    args = parser.parse_args()
    saved = ROOT / '.checks'
    saved.mkdir(exist_ok=True)
    with tempfile.NamedTemporaryFile(prefix='atlas-live-', suffix='.json', dir=saved, delete=False) as handle:
        path = Path(handle.name)
    log = path.with_suffix('.log')

    def get(endpoint):
        result = subprocess.run(['just', 'vm', 'panel', 'GET', endpoint], cwd=ROOT,
                                capture_output=True, text=True, timeout=45)
        with log.open('a') as output:
            output.write(result.stdout + result.stderr)
        if result.returncode:
            raise RuntimeError(f'Atlas API failed; diagnostics: {log}')
        return json.loads(result.stdout)

    status = get('/api/status')
    baseline = args.baseline if args.baseline is None or args.baseline.is_absolute() else ROOT / args.baseline
    before = json.loads(baseline.read_text()) if baseline else None
    live = get('/api/map') if status.get('phase') == 'in_world' else None
    if before:
        key, bounds = before['key'], before['bounds']
        if live and live.get('worldKey') != key:
            raise RuntimeError('Current live world/epoch differs from the baseline')
    else:
        live = live or {}
        key = live.get('worldKey')
        if not key or status.get('phase') != 'in_world':
            raise RuntimeError('Capture requires a live identified world, or --baseline for saved/offline proof')
        x, _, z = status['pos']
        bounds = dict(minX=round(x-128, 3), maxX=round(x+128, 3), minZ=round(z-128, 3), maxZ=round(z+128, 3))
    atlas = get('/api/atlas/map?' + urllib.parse.urlencode(dict(world=key, **bounds)))
    current = samples(atlas.get('tiles', []))
    evidence = dict(at=datetime.datetime.now(datetime.timezone.utc).isoformat(), key=key,
                    bounds=bounds, status=status, atlas=atlas, samples=current)
    path.write_text(json.dumps(evidence, indent=2) + '\n')
    if not current:
        raise RuntimeError(f'No measured atlas samples; evidence: {path}')
    if before:
        previous = before['samples']
        try:
            changed = retained(previous, current)
        except RuntimeError as error:
            raise RuntimeError(f'{error}; evidence: {path}') from error
        print(f'PASS: {len(previous)} baseline samples retained; {changed} changed; no older measurement times')
    print(f"Atlas: {len(current)} canonical samples, {len(atlas['tiles'])} viewport tiles; "
          f"phase={status.get('phase', 'offline')}, hp={status.get('hp', 'n/a')}, version={status.get('version', 'unknown')}")
    print(f'Evidence: {path}')


if __name__ == '__main__':
    try:
        main()
    except (RuntimeError, ValueError, KeyError, OSError, subprocess.TimeoutExpired) as error:
        raise SystemExit(str(error)) from error
