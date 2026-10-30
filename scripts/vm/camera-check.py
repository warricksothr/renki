#!/usr/bin/env python3
"""Verify bounded live camera override cadence; always stop and leave panel online."""
import datetime
import json
from pathlib import Path
import statistics
import subprocess
import time

ROOT = Path(__file__).resolve().parents[2]


def camera_settings(payload):
    # Windows PowerShell may wrap an API array when piping ConvertTo-Json.
    entries = payload.get('value', []) if isinstance(payload, dict) else payload
    return [entry for entry in entries if entry['key'].startswith('screenshot.')]


def cadence(rows, lease):
    """Use actual capture timestamps; never infer delivery cadence from API settings."""
    expiry = datetime.datetime.fromisoformat(lease['expiresAt']).timestamp()
    start = expiry - 15
    groups = {name: [] for name in ('before', 'temporary', 'restored')}
    costs = []
    for row in rows:
        frame = row.get('frame', {})
        if frame.get('type') != 'screenshot':
            continue
        if 'jpeg' in frame or 'jpegBase64' in frame:
            raise RuntimeError('image bytes leaked into audit')
        if row.get('conn') != lease['connection'] or frame.get('worldUid') != lease['worldUid']:
            raise RuntimeError('capture scope changed')
        at = datetime.datetime.fromisoformat(frame['capturedAt']).timestamp()
        stage = 'before' if at < start else 'temporary' if at < expiry else 'restored'
        groups[stage].append(at)
        costs.append(frame['captureMs'])
    summary = {}
    for stage, stamps in groups.items():
        expected = 1 if stage == 'temporary' else 5
        gaps = [b - a for a, b in zip(stamps, stamps[1:])]
        if len(gaps) < (3 if stage == 'temporary' else 1):
            raise RuntimeError('too few actual captures in ' + stage)
        if any(gap < expected * .95 or gap > expected * 1.3 + .15 for gap in gaps):
            raise RuntimeError('unexpected actual capture cadence in ' + stage)
        summary[stage] = dict(frames=len(stamps), medianS=round(statistics.median(gaps), 3),
                              minS=round(min(gaps), 3), maxS=round(max(gaps), 3))
    summary['maxCaptureMs'] = round(max(costs), 2)
    return summary


def main():
    out = ROOT / '.checks' / ('camera-override-live-' + datetime.datetime.now(datetime.timezone.utc).strftime('%Y%m%dT%H%M%S%fZ') + '.json')
    out.parent.mkdir(exist_ok=True)

    def cmd(*args):
        result = subprocess.run(['just', *args], cwd=ROOT, text=True, capture_output=True, timeout=90)
        if result.returncode:
            raise RuntimeError(' '.join(args) + ': ' + result.stderr[-1000:])
        return result.stdout.strip()

    def api(path):
        return json.loads(cmd('vm', 'panel', 'GET', path))

    def screenshot_settings():
        return camera_settings(api('/api/settings'))

    evidence = dict(since=datetime.datetime.now(datetime.timezone.utc).isoformat(), samples=[])
    try:
        state, base = api('/api/status'), api('/api/camera')
        evidence['initial'] = {key: state.get(key) for key in ('version', 'phase', 'connected', 'connection', 'worldUid', 'pos', 'hp', 'hpMax')}
        evidence['base'] = base
        evidence['settingsBefore'] = screenshot_settings()
        if not state.get('connected') or state.get('phase') != 'in_world':
            raise RuntimeError('camera check requires an in-world bot')
        if base['config']['intervalS'] != 5 or not base['config']['enabled'] or base.get('override'):
            raise RuntimeError('requires normal 5-second capture enabled and no override; use just vm camera-rate 5 first')
        scope = (state['connection'], state['worldUid'])
        print('Baseline: same game online, capture enabled at 5 seconds.', flush=True)
        began, lease = time.monotonic(), None
        while time.monotonic() - began < 48:
            elapsed = time.monotonic() - began
            if elapsed >= 11 and lease is None:
                lease = json.loads(cmd('vm', 'camera-diagnostic', '1', '15'))
                evidence['request'] = lease
                if not lease.get('override'):
                    raise RuntimeError('override not active')
                print('Temporary 1-second captures started for 15 seconds.', flush=True)
            current, meta = api('/api/status'), api('/api/camera')
            evidence['samples'].append(dict(elapsedS=round(elapsed, 2), phase=current.get('phase'), connection=current.get('connection'),
                                            worldUid=current.get('worldUid'), hp=current.get('hp'), hpMax=current.get('hpMax'), camera=meta))
            if scope != (current.get('connection'), current.get('worldUid')) or current.get('phase') != 'in_world':
                raise RuntimeError('connection/world changed')
            if current.get('hp', 0) < current.get('hpMax', 1) * .8:
                raise RuntimeError('health below live diagnostic bound')
            time.sleep(1)
        final = api('/api/camera')
        evidence['final'] = final
        if final.get('override') is not None or final['config'] != base['baseConfig'] or final['baseConfig'] != base['baseConfig']:
            raise RuntimeError('base not automatically restored')
        evidence['settingsAfter'] = screenshot_settings()
        if evidence['settingsAfter'] != evidence['settingsBefore']:
            raise RuntimeError('normal camera settings changed during diagnostic')
        evidence['jpeg'] = cmd('vm', 'camera')
        evidence['map'] = cmd('vm', 'map-snapshot')
        evidence['outcome'] = 'passed'
    except Exception as error:
        evidence.update(error=str(error), outcome='failed')
    finally:
        evidence['until'] = datetime.datetime.now(datetime.timezone.utc).isoformat()
        evidence['cleanup'] = {}
        for args in (('vm', 'stop'), ('vm', 'panel-start')):
            try:
                evidence['cleanup'][' '.join(args)] = cmd(*args)
            except Exception as error:
                evidence['cleanup'][' '.join(args)] = str(error)
                evidence['outcome'] = 'failed'
        try:
            evidence['audit'] = cmd('vm', 'audit-window', evidence['since'], evidence['until'])
            if evidence['outcome'] == 'passed':
                rows = [json.loads(line) for line in Path(evidence['audit']).read_text().splitlines()]
                evidence['cadence'] = cadence(rows, evidence['request']['override'])
        except Exception as error:
            evidence.update(auditError=str(error), outcome='failed')
        out.write_text(json.dumps(evidence, indent=2) + '\n')
        print(json.dumps({key: evidence.get(key) for key in ('outcome', 'error', 'auditError', 'cadence')} | dict(evidence=str(out)), indent=2))
    return 0 if evidence['outcome'] == 'passed' else 1


if __name__ == '__main__':
    raise SystemExit(main())
