#!/usr/bin/env python3
"""Supervise a marker-food batch or dry round trip with normal controls and receipts."""
import argparse
import datetime
import json
import math
from pathlib import Path
import subprocess
import time
from harvest_trial import health_loss

ROOT = Path(__file__).resolve().parents[2]


def rows(value):
    # Windows PowerShell can wrap an Invoke-RestMethod array while piping it
    # through ConvertTo-Json; the HTTP API itself returns the plain array.
    return value.get('value', []) if isinstance(value, dict) else value


def unsafe(status, previous=None):
    if not status.get('connected') or status.get('phase') != 'in_world':
        return 'bot unavailable'
    hp, maximum = status.get('hp', 0), status.get('hpMax', 0)
    fraction = .8 if previous is None else .4
    if not math.isfinite(hp) or not math.isfinite(maximum) or maximum <= 0 or hp < max(25, maximum * fraction):
        return 'unsafe health'
    if (status.get('location') or {}).get('inWater') or 'swimming' in status.get('flags', []):
        return 'unsafe footing'
    if previous and ((status.get('connection'), status.get('world')) != (previous.get('connection'), previous.get('world')) or health_loss(previous, status)):
        return 'session changed or actual injury'
    return None


def safe_home(status, mapped, previous=None):
    home = status.get('home') or {}
    perception = mapped.get('perception')
    return (not unsafe(status, previous) and not status.get('task') and 'in_shelter' in status.get('flags', [])
            and home.get('world') == status.get('world') and bool(home.get('pos'))
            and math.dist(status['pos'], home['pos']) <= 8 and abs(status['pos'][1] - home['pos'][1]) <= 1.75
            and (mapped.get('world'), mapped.get('connection')) == (status.get('world'), status.get('connection'))
            and isinstance(perception, dict) and 'characters' in perception
            and not any(c.get('hpFrac', 0) > 0 and (c.get('hostile') or c.get('targetingMe'))
                        and math.dist(c['pos'], status['pos']) < 20 for c in perception['characters']))


def travel_ready(status):
    return any(f.get('remainingS', 0) >= 300 for f in (status.get('food') or {}).get('active', []))


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('marker')
    parser.add_argument('--go', action='store_true', help='move to this marker before starting')
    parser.add_argument('--travel-only', action='store_true', help='verify the outward/home journey without food work; requires --go')
    parser.add_argument('--travel-timeout', type=int, choices=range(120, 181), default=120, help='per-leg travel bound in seconds (120–180)')
    parser.add_argument('--idle', action='store_true', help='verify automatic 30-second idle dwell rather than manual start')
    parser.add_argument('--keep-online', action='store_true', help='remain online only after a verified safe home return')
    args = parser.parse_args()
    if args.travel_only and (not args.go or args.idle):
        parser.error('--travel-only requires --go and cannot use --idle')
    folder = ROOT / '.checks' / ('marker-food-' + datetime.datetime.now(datetime.timezone.utc).strftime('%Y%m%dT%H%M%SZ'))
    folder.mkdir(parents=True)
    summary = dict(outcome='failed', marker=args.marker, evidence=str(folder))
    previous, name, world, saved_definition = None, None, None, None
    owns_body, yielded = False, False
    pinned_home = None
    deadline = time.monotonic() + args.travel_timeout * 3 + 180

    def call(*arguments):
        result = subprocess.run(['just', 'vm', *arguments], cwd=ROOT, capture_output=True, text=True, timeout=45)
        with (folder / 'commands.log').open('a') as log:
            log.write(result.stdout + result.stderr)
        if result.returncode:
            raise RuntimeError('VM request failed: ' + str(arguments[:3]))
        return json.loads(result.stdout)

    def observe():
        nonlocal previous
        status = call('panel', 'GET', '/api/status')
        with (folder / 'observations.jsonl').open('a') as out:
            out.write(json.dumps(dict(at=datetime.datetime.now(datetime.timezone.utc).isoformat(), status=status)) + '\n')
        reason = unsafe(status, previous)
        if reason:
            raise RuntimeError(reason)
        if time.monotonic() > deadline:
            raise RuntimeError('marker-food proof deadline')
        previous = status
        return status

    def task(task, params, timeout):
        nonlocal owns_body, yielded
        request = call('task', task, json.dumps(params), str(timeout))
        owns_body = True
        anchor = datetime.datetime.now(datetime.timezone.utc).isoformat().replace('+00:00', 'Z')
        until = min(deadline, time.monotonic() + timeout + 15)
        seen_running = False
        while time.monotonic() < until:
            status = observe()
            current = status.get('task') or {}
            if current.get('id') == request['id']:
                seen_running = True
            elif seen_running and current.get('id'):
                yielded = True
                raise RuntimeError('new body order interrupted supervised movement')
            history = call('panel', 'GET', '/api/task?id=' + request['id'] + '&at=' + anchor)
            (folder / (request['id'] + '.json')).write_text(json.dumps(history))
            for row in reversed(rows(history)):
                frame = row.get('frame', {})
                if frame.get('type') == 'task' and frame.get('re') == request['id'] and frame.get('status') in ('succeeded', 'failed', 'rejected', 'cancelled'):
                    if frame['status'] != 'succeeded':
                        if frame['status'] == 'cancelled':
                            yielded = True
                        raise RuntimeError('bounded ' + task + ': ' + str(frame))
                    return
            time.sleep(2)
        raise RuntimeError('bounded ' + task + ' deadline')

    try:
        status = observe()
        if status.get('task'):
            raise RuntimeError('body is busy')
        world = status['world']
        home = status.get('home') or {}
        if home.get('world') == world and home.get('pos'):
            pinned_home = list(home['pos'])
        mapped = call('panel', 'GET', '/api/map')
        (folder / 'before-map.json').write_text(json.dumps(mapped))
        markers = [m for m in mapped.get('waypoints', []) if m['name'] == args.marker]
        if len(markers) != 1 or markers[0]['kind'] not in ('berries', 'cooking', 'storage'):
            raise RuntimeError('missing or unsupported food marker')
        marker = markers[0]
        if args.go:
            if not pinned_home or not travel_ready(status):
                raise RuntimeError('outdoor travel needs a pinned home and active meal; resupply first')
            task('move_to', dict(to=marker['pos'], dry=True, avoidHostiles=True), args.travel_timeout)
            status = observe()
        if args.travel_only:
            summary.update(outcome='succeeded', reason='bounded dry round trip completed with native receipts', travelOnly=True)
        else:
            before = call('panel', 'GET', '/api/inventory')
            (folder / 'before-pack.json').write_text(json.dumps(before))
            scope = dict(world=world, name=marker['name'], kind=marker['kind'])
            name = 'marker food check ' + folder.name
            trigger = dict(kind='idle', marker=scope, dwellS=30) if args.idle else dict(kind='manual')
            definition = dict(template='marker_food', params=dict(marker=scope), trigger=trigger,
                              repeat=dict(mode='trigger', cooldownS=300, maxRuns=1) if args.idle else dict(mode='once'), priority=-10, enabled=True)
            call('panel', 'POST', '/api/behaviours', json.dumps(dict(action='save', name=name, definition=definition)))
            saved_definition = definition
            owns_body = True
            if not args.idle:
                call('panel', 'POST', '/api/behaviours', json.dumps(dict(action='start', name=name)))
            started = False
            while time.monotonic() < deadline:
                status = observe()
                instances = rows(call('panel', 'GET', '/api/behaviours'))
                row = next(r for r in instances if r['name'] == name)
                (folder / 'behaviour.json').write_text(json.dumps(row, indent=2))
                if row.get('last') == 'new order':
                    yielded = True
                    raise RuntimeError('new command interrupted marker food work')
                started = started or row['running']
                if row['runs'] > 0:
                    if row.get('last') != 'completed':
                        raise RuntimeError(row.get('last', 'food work did not complete'))
                    break
                if started and not row['running']:
                    yielded = row.get('last') == 'new order'
                    raise RuntimeError('food work interrupted: ' + row.get('last', 'unknown'))
                time.sleep(2)
            else:
                raise RuntimeError('marker-food batch deadline')
            after = call('panel', 'GET', '/api/inventory')
            (folder / 'after-pack.json').write_text(json.dumps(after))
            summary.update(outcome='succeeded', reason='bounded marker food batch completed with native receipts', kind=marker['kind'], idle=args.idle)
        home = status.get('home') or {}
        if home.get('world') != world or home.get('pos') != pinned_home:
            raise RuntimeError('no pinned home for return')
        if not safe_home(status, call('panel', 'GET', '/api/map'), previous):
            task('move_to', dict(to=home['pos'], dry=True, avoidHostiles=True), args.travel_timeout)
        status = observe()
        if not safe_home(status, call('panel', 'GET', '/api/map'), previous):
            raise RuntimeError('return receipt lacks fresh safe home-floor arrival')
    except Exception as error:
        summary.update(outcome='failed', reason=str(error))
    finally:
        if name and saved_definition:
            try:
                call('panel', 'POST', '/api/behaviours', json.dumps(dict(action='disable', name=name)))
            except Exception as error:
                summary.update(outcome='failed', reason='cannot disable test instance: ' + str(error))
        remain = False
        if args.keep_online:
            try:
                status = call('panel', 'GET', '/api/status')
                mapped = call('panel', 'GET', '/api/map')
                remain = safe_home(status, mapped)
                # An empty site is a normal bounded outcome. Return safely
                # rather than leave Tas outside or restart a healthy client.
                if owns_body and not yielded and not remain and not unsafe(status, previous) and not status.get('task') and pinned_home:
                    home = status.get('home') or {}
                    if home.get('world') == world and home.get('pos') == pinned_home:
                        task('move_to', dict(to=pinned_home, dry=True, avoidHostiles=True), args.travel_timeout)
                        status = call('panel', 'GET', '/api/status')
                        mapped = call('panel', 'GET', '/api/map')
                        remain = safe_home(status, mapped)
                # Food production can raise maximum HP without damage.
                # At safe home, allow a bounded natural recovery observation.
                recover_until = min(deadline, time.monotonic() + 90)
                while not remain and not yielded and safe_home(status, mapped, previous) and time.monotonic() < recover_until:
                    time.sleep(2)
                    status = observe()
                    mapped = call('panel', 'GET', '/api/map')
                    remain = safe_home(status, mapped)
            except Exception:
                remain = False
        if owns_body and not yielded and not remain:
            try:
                call('panel', 'POST', '/api/act/logout', '{}')
            except Exception:
                pass
        summary['keptOnline'] = remain
        summary['yieldedToCommand'] = yielded
        (folder / 'summary.json').write_text(json.dumps(summary, indent=2) + '\n')
        print(json.dumps(summary))
    return 0 if summary['outcome'] == 'succeeded' else 1


if __name__ == '__main__':
    raise SystemExit(main())
