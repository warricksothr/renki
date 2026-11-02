#!/usr/bin/env python3
"""Offline checks of the VM lease guard (lease.py) against a local lease file."""
import datetime
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile

LEASE = Path(__file__).with_name('lease.py')
NOW = '2026-11-02T12:00:00Z'


def check(condition, message):
    if not condition:
        raise AssertionError(message)


class Case:
    def __init__(self, root):
        self.root = Path(root)
        self.file = self.root / 'lease.json'

    def env(self, worktree='a', now=NOW):
        env = dict(os.environ, RENKI_VM_LEASE_FILE=str(self.file),
                   RENKI_VM_LEASE_LOCAL=str(self.root / f'{worktree}.vm-lease'))
        env.pop('RENKI_VM_ACTOR', None)
        if now:
            env['RENKI_VM_LEASE_NOW'] = now
        else:
            env.pop('RENKI_VM_LEASE_NOW', None)
        return env

    def lease(self, *args, worktree='a', now=NOW):
        return subprocess.run([sys.executable, '-B', str(LEASE), *args], env=self.env(worktree, now),
                              capture_output=True, text=True, timeout=60)

    def held(self):
        return json.loads(self.file.read_text()) if self.file.exists() else None


def other_holder(case):
    """A guard refuses while another worktree holds an unexpired lease, and says who and until when."""
    check(case.lease('take', 'agent:x/one', '30').returncode == 0, 'take a free lease')
    before = case.file.read_bytes()
    for action in (['guard'], ['take', 'agent:y/two'], ['release'], ['run', '--', 'true']):
        out = case.lease(*action, worktree='b')
        check(out.returncode == 1, f'{action} must be refused: {out}')
        check('agent:x/one' in out.stderr and '2026-11-02T12:30:00Z' in out.stderr,
              f'{action} refusal names holder and expiry: {out.stderr}')
    check(case.file.read_bytes() == before, 'a refused action leaves the lease untouched')
    status = case.lease('status', worktree='b')
    check(status.returncode == 0 and 'held by agent:x/one' in status.stdout, f'status reads: {status.stdout}')


def expired(case):
    """An expired lease is free: the next guard takes it."""
    check(case.lease('take', 'agent:x/one', '30').returncode == 0, 'take')
    later = '2026-11-02T12:31:00Z'
    check('free (expired' in case.lease('status', worktree='b', now=later).stdout, 'status says expired')
    out = case.lease('guard', worktree='b', now=later)
    check(out.returncode == 0 and 'took the VM lease' in out.stderr, f'guard takes an expired lease: {out}')
    held = case.held()
    check(held['expires'] == '2026-11-02T13:31:00Z', f'default 60 minutes from the VM clock: {held}')
    check(case.lease('guard', worktree='a', now=later).returncode == 1, 'the old holder is now refused')


def held_by_self(case):
    """The holder passes the guard without a write, renews, and releases."""
    check(case.lease('take', 'agent:x/one', '30').returncode == 0, 'take')
    before = case.file.read_bytes()
    out = case.lease('guard')
    check(out.returncode == 0 and out.stderr == '', f'guard passes quietly for the holder: {out}')
    check(case.file.read_bytes() == before, 'guard does not rewrite a held lease')
    first = case.held()
    out = case.lease('take', 'agent:x/one', '90', now='2026-11-02T12:20:00Z')
    check(out.returncode == 0 and out.stdout.startswith('renewed'), f'renew: {out}')
    renewed = case.held()
    check(renewed['id'] == first['id'] and renewed['started'] == first['started'], 'renewal keeps the lease')
    check(renewed['expires'] == '2026-11-02T13:50:00Z', f'renewal moves the expiry: {renewed}')
    out = case.lease('release')
    check(out.returncode == 0 and case.held() is None, f'release removes it: {out}')
    check(not (case.root / 'a.vm-lease').exists(), 'release forgets the local id')
    check(case.lease('guard', worktree='b').returncode == 0, 'anyone may take a released lease')


def malformed(case):
    """An unreadable lease file blocks everyone until the owner clears it."""
    for content in (b'{not json', b'[]', b'{"id": "x", "actor": "y", "expires": "soon"}', b'\xff\xfe'):
        case.file.write_bytes(content)
        out = case.lease('guard')
        check(out.returncode == 1 and 'unreadable' in out.stderr, f'guard refuses {content!r}: {out}')
        check(case.lease('release').returncode == 1, 'plain release refuses an unreadable lease')
        check(case.file.read_bytes() == content, 'refusal leaves the file')
    out = case.lease('release', '--force')
    check(out.returncode == 0 and case.held() is None, f'release --force clears it: {out}')


def forced_release(case):
    """--force removes someone else's lease and says whose it was."""
    check(case.lease('take', 'agent:x/one').returncode == 0, 'take')
    out = case.lease('release', '--force', worktree='b')
    check(out.returncode == 0 and 'agent:x/one' in out.stdout and case.held() is None, f'forced: {out}')


def run_releases_after_cleanup(case):
    """run holds the lease through the command's own cleanup and releases it afterwards."""
    probe = case.root / 'probe.json'
    child = (
        'import json, os, pathlib, sys\n'
        'lease = pathlib.Path(sys.argv[1])\n'
        'try:\n'
        '    seen = dict(during=lease.exists())\n'
        '    sys.exit(3)\n'
        'finally:\n'
        '    seen["cleanup"] = lease.exists()\n'
        '    pathlib.Path(sys.argv[2]).write_text(json.dumps(seen))\n'
    )
    out = case.lease('run', '--', sys.executable, '-c', child, str(case.file), str(probe))
    check(out.returncode == 3, f"run returns the command's status: {out}")
    seen = json.loads(probe.read_text())
    check(seen == dict(during=True, cleanup=True), f'held during the command and its cleanup: {seen}')
    check(case.held() is None and 'released the VM lease' in out.stderr, f'released afterwards: {out}')


def run_keeps_a_taken_lease(case):
    """A lease taken with `take` before a trial stays with its holder afterwards."""
    check(case.lease('take', 'agent:x/one').returncode == 0, 'take')
    out = case.lease('run', '--', 'true')
    check(out.returncode == 0 and case.held() is not None, f'run leaves a lease it did not take: {out}')


def run_renews(case):
    """run renews the lease while a long command runs, on the VM clock."""
    check(case.lease('take', 'agent:x/one', '120', now=None).returncode == 0, 'take a lease')
    # The child stands in for time passing: it shortens the lease to a minute
    # after run started, so only a renewal inside the loop can lengthen it.
    child = (
        'import json, pathlib, sys, time\n'
        'path = pathlib.Path(sys.argv[1])\n'
        'lease = json.loads(path.read_text())\n'
        'lease["expires"] = sys.argv[2]\n'
        'path.write_text(json.dumps(lease))\n'
        'time.sleep(0.4)\n'
    )
    short = (datetime.datetime.now(datetime.timezone.utc) + datetime.timedelta(minutes=1)).strftime('%Y-%m-%dT%H:%M:%SZ')
    out = case.lease('run', '--minutes', '90', '--renew-seconds', '0.05', '--',
                     sys.executable, '-c', child, str(case.file), short, now=None)
    check(out.returncode == 0, f'run: {out}')
    check(case.held()['expires'] > short, f'renewed during the command beyond {short}: {case.held()}')


def run_survives_a_failed_renewal(case):
    """A renewal that fails is a warning; run still waits for the command and its cleanup."""
    probe = case.root / 'done'
    # A directory in place of the lease file makes every read fail with an
    # error that is not a refusal, as a broken SSH reply would.
    child = (
        'import pathlib, sys, time\n'
        'lease, probe = pathlib.Path(sys.argv[1]), pathlib.Path(sys.argv[2])\n'
        'lease.unlink(); lease.mkdir(); time.sleep(0.3); lease.rmdir()\n'
        'probe.write_text("cleanup finished")\n'
    )
    check(case.lease('take', 'agent:x/one').returncode == 0, 'take')
    out = case.lease('run', '--renew-seconds', '0.05', '--', sys.executable, '-c', child, str(case.file), str(probe))
    check(out.returncode == 0 and probe.exists(), f'run waited for the command: {out}')
    check('could not renew' in out.stderr, f'the failed renewal is reported: {out.stderr}')


def lost_race(case):
    """A write against a stale read is refused, so two takers cannot both win."""
    sys.path.insert(0, str(LEASE.parent))
    import importlib
    lease = importlib.import_module('lease')
    transport = lease.FileTransport(case.file)
    check(transport.write('', b'{}'), 'first write of an absent file')
    check(not transport.write('', b'{}'), 'second write with a stale hash is refused')


def main():
    cases = [other_holder, expired, held_by_self, malformed, forced_release,
             run_releases_after_cleanup, run_keeps_a_taken_lease, run_renews, run_survives_a_failed_renewal,
             lost_race]
    for case in cases:
        with tempfile.TemporaryDirectory() as root:
            case(Case(root))
    print(f'lease: {len(cases)} cases passed')


if __name__ == '__main__':
    main()
