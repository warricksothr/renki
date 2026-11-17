#!/usr/bin/env python3
"""Verify real viewer permissions without logging credentials or changing game work.

Temporarily enable sharing only if disabled, then restore disabled state. Existing
links are preserved. Established WebRTC media is not forcibly disconnected.
"""
import datetime
import http.cookiejar
import json
from pathlib import Path
import subprocess
import urllib.error
import urllib.request

ROOT = Path(__file__).resolve().parents[2]


def recipe(*args):
    result = subprocess.run(['just', *args], cwd=ROOT, text=True,
                            capture_output=True, timeout=45)
    if result.returncode:
        raise RuntimeError('Panel tooling failed; credential output withheld')
    return result.stdout.strip()


def admin(method, path, body=None):
    args = ['vm', 'panel', method, path]
    if body is not None:
        args.append(json.dumps(body))
    return json.loads(recipe(*args))


def main():
    address = recipe('vm', 'panel-address').rstrip('/')
    if not address.startswith('http://') and not address.startswith('https://'):
        raise RuntimeError('Panel address unavailable')
    initial = admin('GET', '/api/sharing')
    temporary = not initial['enabled']
    summary = {'at': datetime.datetime.now(datetime.timezone.utc).isoformat(),
               'checks': [], 'initiallyEnabled': not temporary}
    path = ROOT / '.checks' / 'viewer-live.json'
    path.parent.mkdir(exist_ok=True)
    # No environment proxy may receive the private address or cookies.
    opener = urllib.request.build_opener(urllib.request.ProxyHandler({}),
                                        urllib.request.HTTPCookieProcessor(http.cookiejar.CookieJar()))

    def request(method, endpoint, body=None):
        data = json.dumps(body).encode() if body is not None else None
        req = urllib.request.Request(address + endpoint, data=data, method=method,
                                     headers={'Content-Type': 'application/json', 'X-Renki-Act': 'viewer-check'})
        try:
            response = opener.open(req, timeout=10)
        except urllib.error.HTTPError as error:
            response = error
        with response:
            if endpoint.endswith('/events') and response.status == 200:
                payload = response.readline() + response.readline()
            else:
                payload = response.read(32 << 20)
            return response.status, payload

    def expect(method, endpoint, status, body=None):
        code, payload = request(method, endpoint, body)
        if code != status:
            raise RuntimeError(f'{method} {endpoint}: expected {status}, got {code}')
        summary['checks'].append(f'{method} {endpoint}: {code}')
        return payload

    try:
        current = admin('POST', '/api/sharing', {'action': 'rotate'}) if temporary else initial
        expect('POST', '/api/viewer/login', 204, {'token': current['token']})
        status = json.loads(expect('GET', '/api/status', 200))
        private = {'owners', 'players', 'presence', 'model', 'goals', 'foodDecision', 'homeRules', 'captureProof'}
        if private.intersection(status):
            raise RuntimeError('Private status fields exposed')
        mapped = json.loads(expect('GET', '/api/map', 200))
        if mapped.get('containers'):
            raise RuntimeError('Chest contents exposed')
        stream = expect('GET', '/api/viewer/events', 200)
        if not stream.startswith(b'event: status\n') or b'event: record' in stream:
            raise RuntimeError('Viewer stream is not observation-only')
        expect('GET', '/api/video', 200)
        expect('GET', '/api/atlas/worlds', 200)
        for endpoint in ['/api/settings', '/api/inventory', '/api/timeline', '/api/inspection', '/api/memory/players', '/api/sharing']:
            expect('GET', endpoint, 403)
        for endpoint in ['/api/act/stop', '/api/say', '/api/task', '/api/setting', '/api/goals', '/api/behaviours', '/api/waypoints', '/api/bases']:
            expect('POST', endpoint, 403, {})
        if temporary:
            admin('POST', '/api/sharing', {'action': 'rotate'})
            expect('GET', '/api/status', 401)
        summary['status'] = 'passed'
    finally:
        if temporary:
            admin('POST', '/api/sharing', {'action': 'disable'})
        summary['finalEnabled'] = admin('GET', '/api/sharing')['enabled']
        path.write_text(json.dumps(summary, indent=2) + '\n')
    print(f'Viewer permissions passed; sharing restored; evidence: {path}')


if __name__ == '__main__':
    main()
