#!/usr/bin/env python3
"""Local-only MediaMTX proof tooling. No VM or external output support."""
import argparse
import hashlib
import http.server
import json
import os
from pathlib import Path
import secrets
import subprocess
import tarfile
import urllib.request

ROOT = Path(__file__).resolve().parents[2]
OUT = ROOT / '.checks/streaming'
VERSION = '1.21.3'
DIGEST = '0a2ea7eba96023631efa1bea7804823a9145b043b6646a8a05798d4ac24ed635'


def configuration(password, slate):
    # JSON is a YAML subset accepted by MediaMTX. All keys verified against
    # v1.21.3/mediamtx.yml. No default catch-all path or public listener.
    return dict(logLevel='warn', api=False, metrics=False, pprof=False,
                playback=False, hls=False, srt=False, moq=False,
                rtsp=True, rtspAddress='127.0.0.1:18554', rtspTransports=['tcp'],
                rtmp=True, rtmpAddress='127.0.0.1:11935',
                webrtc=True, webrtcAddress='127.0.0.1:18889',
                webrtcLocalUDPAddress='127.0.0.1:18189',
                webrtcLocalTCPAddress='127.0.0.1:18189',
                webrtcIPsFromInterfaces=False, webrtcAdditionalHosts=['127.0.0.1'],
                webrtcICEServers2=[], webrtcAllowOrigins=[],
                authInternalUsers=[
                    dict(user='any', ips=['127.0.0.1'], permissions=[dict(action='publish', path='renki')]),
                    dict(user='panel', **{'pass': password}, ips=['127.0.0.1'], permissions=[dict(action='read', path='renki')])],
                paths=dict(renki=dict(source='publisher', maxReaders=4, overridePublisher=False,
                                     alwaysAvailable=True, alwaysAvailableFile=str(slate),
                                     record=False, forward=[])))


def prepare():
    OUT.mkdir(parents=True, exist_ok=True)
    # Preserve the random server-side read credential across invocations.
    path = OUT / 'panel.local.json'
    if path.exists():
        password = json.loads(path.read_text())['password']
    else:
        password = secrets.token_urlsafe(32)
    for path, value in [(path, dict(port=18889, password=password, healthFile=str(OUT / 'health.local.json'))),
                        (OUT / 'mediamtx.local.yml', configuration(password, OUT / 'standby.mp4'))]:
        # Never truncate an existing file before securing its mode.
        fd = os.open(path, os.O_WRONLY | os.O_CREAT, 0o600)
        os.fchmod(fd, 0o600)
        os.ftruncate(fd, 0)
        with os.fdopen(fd, 'w') as stream:
            json.dump(value, stream, indent=2)
            stream.write('\n')
    print('Private relay config prepared in .checks/streaming (loopback only)')


def install():
    OUT.mkdir(parents=True, exist_ok=True)
    archive = OUT / 'mediamtx.tar.gz'
    url = f'https://github.com/bluenviron/mediamtx/releases/download/v{VERSION}/mediamtx_v{VERSION}_linux_amd64.tar.gz'
    with urllib.request.urlopen(url, timeout=30) as response:
        data = response.read(40 * 1024 * 1024)
    if hashlib.sha256(data).hexdigest() != DIGEST:
        raise SystemExit('MediaMTX release checksum mismatch')
    archive.write_bytes(data)
    with tarfile.open(archive) as tar:
        member = tar.getmember('mediamtx')
        if not member.isfile():
            raise SystemExit('Invalid release executable')
        binary = OUT / 'mediamtx'
        binary.write_bytes(tar.extractfile(member).read())
        binary.chmod(0o700)
    print(f'Installed checksum-verified MediaMTX v{VERSION} in .checks/streaming')


def slate():
    data = (ROOT / 'internal/panel/static/slate.html').read_bytes()

    class Handler(http.server.BaseHTTPRequestHandler):
        def do_GET(self):
            if self.path != '/':
                self.send_error(404)
                return
            self.send_response(200)
            self.send_header('Content-Type', 'text/html; charset=utf-8')
            self.send_header('Content-Length', str(len(data)))
            self.send_header('Content-Security-Policy', "default-src 'none'; script-src 'unsafe-inline'; style-src 'unsafe-inline'; frame-ancestors 'none'")
            self.end_headers()
            self.wfile.write(data)

        def log_message(self, *_):
            pass

    print('Original OBS slate: http://127.0.0.1:18794/ (no control/API routes)', flush=True)
    http.server.HTTPServer(('127.0.0.1', 18794), Handler).serve_forever()


def source(vp9=False):
    os.execvp('ffmpeg', ['ffmpeg', '-hide_banner', '-loglevel', 'error', '-re',
                        '-f', 'lavfi', '-i', 'testsrc2=size=1280x720:rate=30',
                        '-an', *(['-c:v', 'libvpx-vp9', '-deadline', 'realtime', '-cpu-used', '8', '-row-mt', '1'] if vp9 else ['-c:v', 'libx264', '-preset', 'ultrafast', '-tune', 'zerolatency', '-profile:v', 'baseline']), '-pix_fmt', 'yuv420p', '-g', '60', '-bf', '0',
                        '-b:v', '3000k', '-strict', 'experimental', '-f', 'rtsp', '-rtsp_transport', 'tcp',
                        'rtsp://127.0.0.1:18554/renki'])


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('action', choices=['install', 'prepare', 'relay', 'source', 'source-vp9', 'slate', 'proof-state'])
    parser.add_argument('state', nargs='?', choices=['offline','online','source-on','source-off'])
    args = parser.parse_args()
    if args.action == 'proof-state':
        if not args.state:
            parser.error('proof-state needs a state')
        state = args.state.replace('source-', 'source/')
        request = urllib.request.Request('http://127.0.0.1:18793/proof/' + state, method='POST', headers={'X-Renki-Proof':'1'})
        with urllib.request.urlopen(request, timeout=3) as response:
            print('Synthetic proof state updated:', args.state, response.status)
    elif args.action == 'source-vp9':
        source(vp9=True)
    elif args.action == 'relay':
        os.execv(str(OUT / 'mediamtx'), [str(OUT / 'mediamtx'), str(OUT / 'mediamtx.local.yml')])
    else:
        globals()[args.action]()
