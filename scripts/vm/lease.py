#!/usr/bin/env python3
"""The VM lease: one session at a time deploys to the VM or runs a live trial.

The lease is C:/renki/lease.json on the VM, because the VM is the shared
resource. lease.ps1 reads it and swaps it atomically; every decision is made
here so that it can be tested offline (lease-test.py). docs/VM.md, "The VM
lease", is the operator's guide.

    status                 print the holder, without changing anything
    take ACTOR [MINUTES]   take the lease, or renew it when this worktree holds it
    release [--force]      give it back; --force removes another holder's lease
    guard                  pass when this worktree holds the lease, take it when
                           it is free or expired, refuse when someone else holds it
    run -- COMMAND...      guard, run COMMAND while renewing the lease, then
                           release it after COMMAND exits if run took it

A worktree recognises its own lease by the random id it keeps in .vm-lease at
the repository root (gitignored). Expiry is measured on the VM's clock.
"""
import argparse
import base64
import datetime
import fcntl
import hashlib
import json
import os
from pathlib import Path
import secrets
import signal
import socket
import subprocess
import sys

ROOT = Path(__file__).resolve().parents[2]
DEFAULT_MINUTES = 60
RENEW_SECONDS = 300
EXIT_REFUSED = 1


class Refused(Exception):
    """Someone else holds the lease, or the lease file cannot be read."""


def iso(moment):
    return moment.astimezone(datetime.timezone.utc).strftime('%Y-%m-%dT%H:%M:%SZ')


def parse_time(text):
    moment = datetime.datetime.fromisoformat(text.replace('Z', '+00:00'))
    if moment.tzinfo is None:
        raise ValueError('time without a zone')
    return moment


class SSHTransport:
    """The real lease file, through lease.ps1 over SSH."""

    def __init__(self, target, vm_dir):
        if not target:
            raise SystemExit('no VM: set RENKI_VM=user@host or write it to vm.local')
        if "'" in vm_dir:
            raise SystemExit('RENKI_VM_DIR must not contain a quote')
        self.target = target
        self.vm_dir = vm_dir
        self.script = (Path(__file__).with_name('lease.ps1')).read_text()

    def _run(self, action, expect='', content=''):
        preamble = f"$Dir='{self.vm_dir}'; $Action='{action}'; $Expect='{expect}'; $Content='{content}'\n"
        encoded = base64.b64encode((preamble + self.script).encode('utf-16-le')).decode()
        command = 'powershell -NoProfile -NonInteractive -ExecutionPolicy Bypass -EncodedCommand ' + encoded
        try:
            return subprocess.run(['ssh', '-o', 'BatchMode=yes', '-o', 'ConnectTimeout=10', self.target, command],
                                  capture_output=True, text=True, timeout=90)
        except subprocess.TimeoutExpired:
            raise SystemExit(f'the VM lease {action} timed out after 90 s') from None

    def read(self):
        result = self._run('Read')
        if result.returncode != 0:
            raise SystemExit(f'cannot read the VM lease (SSH exit {result.returncode}): {result.stderr.strip()[-400:]}')
        try:
            reply = json.loads(result.stdout.strip().splitlines()[-1])
            content = base64.b64decode(reply['content'], validate=True) if reply['content'] else None
            return parse_time(reply['now']), reply['sha256'], content
        except (ValueError, IndexError, KeyError, TypeError, AttributeError) as error:
            raise SystemExit(f'cannot parse the VM lease reply: {error}') from None

    def write(self, expect, data):
        content = base64.b64encode(data).decode() if data is not None else ''
        result = self._run('Write', expect, content)
        if result.returncode == 3:
            return False
        if result.returncode != 0:
            raise SystemExit(f'cannot write the VM lease (SSH exit {result.returncode}): '
                             f'{(result.stdout + result.stderr).strip()[-400:]}')
        return True


class FileTransport:
    """A local lease file standing in for the VM in offline tests
    (RENKI_VM_LEASE_FILE). RENKI_VM_LEASE_NOW fixes the clock."""

    def __init__(self, path):
        self.path = Path(path)

    def _now(self):
        fixed = os.environ.get('RENKI_VM_LEASE_NOW')
        return parse_time(fixed) if fixed else datetime.datetime.now(datetime.timezone.utc)

    def _state(self):
        if not self.path.exists():
            return '', None
        data = self.path.read_bytes()
        return hashlib.sha256(data).hexdigest(), data

    def read(self):
        sha, data = self._state()
        return self._now().replace(microsecond=0), sha, data

    def write(self, expect, data):
        with open(str(self.path) + '.lock', 'a') as lock:
            fcntl.flock(lock, fcntl.LOCK_EX)
            if self._state()[0] != expect:
                return False
            if data is None:
                self.path.unlink(missing_ok=True)
            else:
                tmp = Path(str(self.path) + '.tmp')
                tmp.write_bytes(data)
                os.replace(tmp, self.path)
            return True


def classify(data, now, my_id):
    """free, malformed, expired, self or other, and the parsed lease."""
    if data is None:
        return 'free', None
    try:
        lease = json.loads(data.decode('utf-8-sig'))
        if not isinstance(lease, dict) or not all(isinstance(lease.get(k), str) and lease[k]
                                                  for k in ('id', 'actor', 'expires')):
            raise ValueError('missing id, actor or expires')
        expires = parse_time(lease['expires'])
    except (ValueError, UnicodeDecodeError):
        return 'malformed', None
    if expires <= now:
        return 'expired', lease
    if my_id and secrets.compare_digest(lease['id'], my_id):
        return 'self', lease
    return 'other', lease


def describe(lease, now):
    expires = parse_time(lease['expires'])
    left = int((expires - now).total_seconds() // 60)
    where = ', '.join(f'{k} {lease[k]}' for k in ('branch', 'worktree') if lease.get(k))
    return (f"{lease['actor']}" + (f' ({where})' if where else '') +
            f" until {lease['expires']} ({left} min left)")


class Lease:
    def __init__(self, transport, local_path):
        self.transport = transport
        self.local_path = Path(local_path)

    def local(self):
        try:
            value = json.loads(self.local_path.read_text())
            return value if isinstance(value, dict) else {}
        except (OSError, ValueError):
            return {}

    def actor(self, explicit=None):
        user = os.environ.get('USER') or 'someone'
        return (explicit or os.environ.get('RENKI_VM_ACTOR') or self.local().get('actor')
                or f'{user}@{socket.gethostname()}')

    def read(self):
        now, sha, data = self.transport.read()
        state, lease = classify(data, now, self.local().get('id'))
        return now, sha, state, lease

    def refusal(self, state, lease, now):
        if state == 'malformed':
            return ('the VM lease file is unreadable; check `just vm lease-status`, and only the owner '
                    'should clear it with `just vm release --force`')
        return (f'the VM is leased to {describe(lease, now)}; wait for it to expire, or ask them to run '
                '`just vm release`')

    def take(self, actor=None, minutes=DEFAULT_MINUTES, renew='always'):
        """Take the lease, or renew it when this worktree holds it: 'always',
        'extend' (only if that moves the expiry later) or 'never'. Returns
        (lease, took). Raises Refused."""
        for _ in range(5):
            now, sha, state, lease = self.read()
            if state in ('other', 'malformed'):
                raise Refused(self.refusal(state, lease, now))
            expires = now + datetime.timedelta(minutes=minutes)
            if state == 'self':
                if renew == 'never' or (renew == 'extend' and parse_time(lease['expires']) >= expires):
                    return lease, False
                new = dict(lease, expires=iso(expires))
                if actor:
                    new['actor'] = actor
            else:
                new = dict(id=secrets.token_hex(16), actor=self.actor(actor), host=socket.gethostname(),
                           worktree=str(ROOT), branch=branch(), started=iso(now), expires=iso(expires))
            data = (json.dumps(new, indent=2) + '\n').encode()
            if self.transport.write(sha, data):
                self.local_path.write_text(json.dumps(dict(id=new['id'], actor=new['actor'])) + '\n')
                return new, state != 'self'
        raise Refused('the VM lease kept changing while it was being taken; try again')

    def release(self, force=False):
        """Returns a message. Raises Refused."""
        for _ in range(5):
            now, sha, state, lease = self.read()
            if state in ('other', 'malformed') and not force:
                raise Refused(self.refusal(state, lease, now) + '; it is not yours to release')
            if state == 'free':
                self.local_path.unlink(missing_ok=True)
                return 'the VM lease is free'
            if self.transport.write(sha, None):
                self.local_path.unlink(missing_ok=True)
                if state == 'malformed':
                    return 'removed an unreadable VM lease'
                if state == 'expired':
                    return f"removed an expired lease ({lease['actor']}, {lease['expires']})"
                if state == 'other':
                    return f'removed the lease of {describe(lease, now)}'
                return 'released the VM lease'
        raise Refused('the VM lease kept changing while it was being released; try again')

    def release_if_self(self):
        for _ in range(5):
            now, sha, state, lease = self.read()
            if state != 'self':
                return False
            if self.transport.write(sha, None):
                self.local_path.unlink(missing_ok=True)
                return True
        return False


def branch():
    try:
        out = subprocess.run(['git', 'rev-parse', '--abbrev-ref', 'HEAD'], cwd=ROOT,
                             capture_output=True, text=True, timeout=10)
        return out.stdout.strip() if out.returncode == 0 else ''
    except (OSError, subprocess.SubprocessError):
        return ''


def run(lease, command, minutes, renew_seconds):
    """Hold the lease for COMMAND. The command does its own cleanup (combat off,
    stop or logout) before it exits; the lease is released only after that."""
    held, took = lease.take(minutes=minutes, renew='extend')
    if took:
        print(f"took the VM lease for {held['actor']} until {held['expires']}", file=sys.stderr)
    # Ctrl-C reaches the child too; wait for its cleanup instead of dying first.
    previous = signal.signal(signal.SIGINT, signal.SIG_IGN)
    try:
        child = subprocess.Popen(command, preexec_fn=lambda: signal.signal(signal.SIGINT, signal.SIG_DFL))
        while True:
            try:
                code = child.wait(timeout=renew_seconds)
                break
            except subprocess.TimeoutExpired:
                # Any failure to renew is a warning: the trial keeps its lease
                # until expiry and is still waited for, never abandoned.
                try:
                    lease.take(minutes=minutes, renew='extend')
                except (SystemExit, Exception) as error:
                    print(f'could not renew the VM lease: {error}', file=sys.stderr)
    finally:
        signal.signal(signal.SIGINT, previous)
    if took:
        try:
            released = lease.release_if_self()
        except (SystemExit, Exception) as error:
            print(f'could not release the VM lease, which expires on its own: {error}', file=sys.stderr)
        else:
            print('released the VM lease' if released else 'the VM lease was no longer ours to release',
                  file=sys.stderr)
    return code


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument('--target', default=os.environ.get('RENKI_VM', ''))
    parser.add_argument('--dir', default=os.environ.get('RENKI_VM_DIR', 'C:/renki'))
    sub = parser.add_subparsers(dest='action', required=True)
    sub.add_parser('status')
    take = sub.add_parser('take')
    take.add_argument('actor')
    take.add_argument('minutes', nargs='?', type=int, default=DEFAULT_MINUTES)
    release = sub.add_parser('release')
    release.add_argument('--force', action='store_true')
    sub.add_parser('guard')
    runner = sub.add_parser('run')
    runner.add_argument('--minutes', type=int, default=DEFAULT_MINUTES)
    runner.add_argument('--renew-seconds', type=float, default=RENEW_SECONDS)
    runner.add_argument('command', nargs=argparse.REMAINDER)
    args = parser.parse_args(argv)

    if os.environ.get('RENKI_VM_LEASE_FILE'):
        transport = FileTransport(os.environ['RENKI_VM_LEASE_FILE'])
    else:
        transport = SSHTransport(args.target, args.dir)
    lease = Lease(transport, os.environ.get('RENKI_VM_LEASE_LOCAL', ROOT / '.vm-lease'))

    try:
        if args.action == 'status':
            now, _, state, held = lease.read()
            if state == 'free':
                print('the VM lease is free')
            elif state == 'malformed':
                print('the VM lease file is unreadable; the owner can clear it with `just vm release --force`')
            elif state == 'expired':
                print(f"the VM lease is free (expired: {held['actor']}, {held['expires']})")
            else:
                print(('held by this worktree: ' if state == 'self' else 'held by ') + describe(held, now))
            return 0
        if args.action == 'take':
            if not 1 <= args.minutes <= 24 * 60:
                raise SystemExit('MINUTES must be 1 to 1440')
            held, took = lease.take(args.actor, args.minutes)
            print(f"{'took' if took else 'renewed'} the VM lease for {held['actor']} until {held['expires']}")
            return 0
        if args.action == 'release':
            print(lease.release(args.force))
            return 0
        if args.action == 'guard':
            held, took = lease.take(minutes=DEFAULT_MINUTES, renew='never')
            if took:
                print(f"took the VM lease for {held['actor']} until {held['expires']}; "
                      '`just vm release` when you are finished', file=sys.stderr)
            return 0
        command = args.command[1:] if args.command[:1] == ['--'] else args.command
        if not command:
            raise SystemExit('run needs a command after --')
        return run(lease, command, args.minutes, args.renew_seconds)
    except Refused as error:
        print(f'refused: {error}', file=sys.stderr)
        return EXIT_REFUSED


if __name__ == '__main__':
    sys.exit(main())
