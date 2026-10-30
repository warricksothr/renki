#!/usr/bin/env python3
"""One owner-independent melee trial through normal tasks, with retained evidence."""
import argparse
from collections import Counter
import datetime
import json
import math
from pathlib import Path
import subprocess
import sys
import time

ROOT = Path(__file__).resolve().parents[2]


def distance(a, b):
    return math.dist(a, b)


def abort_reason(status, origin=None, characters=(), target=None):
    if not status.get('connected') or status.get('phase') != 'in_world':
        return 'not in the world'
    if status.get('hpMax', 0) <= 0 or status.get('hp', 0) < status['hpMax'] * .7:
        return 'health below trial floor'
    if status.get('location', {}).get('inWater') or 'swimming' in status.get('flags', []):
        return 'unsafe water'
    if origin and distance(origin, status['pos']) > 40:
        return 'trial travel limit'
    if target and any(c['id'] != target and c.get('kind') == 'creature'
                      and c.get('hostile') and c.get('hpFrac', 0) > 0
                      and distance(c['pos'], status['pos']) < 8 for c in characters):
        return 'additional nearby creature'
    if (status.get('combat') or {}).get('mode') == 'retreat' and status.get('stamina', 0) < 10:
        return 'escape stamina exhausted'
    return None


def records(value):
    return value.get('value', []) if isinstance(value, dict) else value


def terminal(value, task_id):
    for row in reversed(records(value)):
        frame = row.get('frame', {})
        if frame.get('re') == task_id and frame.get('status') in ('succeeded', 'failed', 'rejected', 'cancelled'):
            return frame
    return None


def ground_diagnostics(audit, task_id):
    """Bounded projection of one native goal; full traces remain in the audit."""
    reasons, stages = Counter(), Counter()
    examples, seen = [], set()
    with Path(audit).open() as lines:
        for line in lines:
            row = json.loads(line)
            frame = row.get('frame', {})
            if (frame.get('type') != 'trace' or frame.get('task') != task_id
                    or frame.get('source') != 'combat' or frame.get('msg') != 'ground_rejected'):
                continue
            data = frame.get('data') or {}
            reasons[data.get('reason', 'unknown')] += 1
            stage = data.get('probeStage')
            if stage:
                stages[stage] += 1
            kind = (data.get('reason'), stage)
            if kind not in seen and len(examples) < 6:
                seen.add(kind)
                examples.append({k: data[k] for k in ('candidate', 'reason', 'segment', 'from', 'desired',
                                'supported', 'angle', 'previousAngle', 'roughness', 'probeStage', 'probeAt', 'collider',
                                'colliderCenter') if k in data})
    return dict(candidates=sum(reasons.values()), reasons=dict(reasons), probeStages=dict(stages), examples=examples)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--entity', help='optional exact target; otherwise C# acquires a local ordinary Greyling')
    parser.add_argument('--wait-s', type=float, default=30, help='bounded food/stamina/target preparation, 0..120 seconds')
    parser.add_argument('--at', type=float, nargs=3, help='optional preferred nearby fighting-ground center')
    args = parser.parse_args()
    if args.at and not all(math.isfinite(v) for v in args.at):
        parser.error('--at must contain finite coordinates')
    if not math.isfinite(args.wait_s) or not 0 <= args.wait_s <= 120:
        parser.error('--wait-s must be between 0 and 120')
    saved = ROOT / '.checks' / ('combat-trial-' + datetime.datetime.now(datetime.timezone.utc).strftime('%Y%m%dT%H%M%S%fZ'))
    saved.mkdir(parents=True)
    summary = {'outcome': 'not_started', 'evidence': str(saved)}
    since = datetime.datetime.now(datetime.timezone.utc).isoformat().replace('+00:00', 'Z')

    def call(*arguments, timeout=45, as_json=False):
        p = subprocess.run(['just', 'vm', *arguments], cwd=ROOT, capture_output=True, text=True, timeout=timeout)
        with (saved / 'commands.log').open('a') as f:
            f.write('just vm ' + ' '.join(arguments) + '\n' + p.stdout + p.stderr)
        if p.returncode:
            raise RuntimeError('just vm ' + arguments[0] + ' failed; see commands.log')
        return json.loads(p.stdout) if as_json else p.stdout

    def observe():
        status = call('panel', 'GET', '/api/status', as_json=True)
        mapped = call('panel', 'GET', '/api/map', as_json=True)
        with (saved / 'observations.jsonl').open('a') as f:
            f.write(json.dumps({'at': datetime.datetime.now(datetime.timezone.utc).isoformat(), 'status': status,
                                'perception': mapped.get('perception')}) + '\n')
        if status.get('hp') is not None:
            summary['minimumHP'] = min(summary.get('minimumHP', status['hp']), status['hp'])
        summary['lastStatus'] = status
        return status, mapped

    try:
        print('Preparing one bounded combat trial; evidence: ' + str(saved), flush=True)
        call('stop', timeout=200)
        call('combat', 'true')
        call('start', timeout=330)
        deadline = time.monotonic() + args.wait_s
        while True:
            status, mapped = observe()
            reason = abort_reason(status)
            if reason:
                raise RuntimeError(reason)
            chars = (mapped.get('perception') or {}).get('characters', [])
            # Do not wait for regeneration while being chased, or replace an escape.
            if status.get('task'):
                raise RuntimeError('another task is active; trial will not replace it')
            ready = (status.get('hp', 0) >= status.get('hpMax', 1) * .8
                     and status.get('stamina', 0) >= 40 and status.get('food', {}).get('active'))
            if ready:
                break
            if any(c.get('hostile') and c.get('kind') == 'creature' and c.get('hpFrac', 0) > 0
                   and distance(c['pos'], status['pos']) < 15 for c in chars):
                raise RuntimeError('cannot prepare safely near a hostile creature')
            if time.monotonic() >= deadline:
                raise RuntimeError('no fed, rested, safe single-Greyling encounter available')
            time.sleep(1)
        goal = {'kite': True, 'style': 'melee'}
        if args.entity:
            goal['entity'] = args.entity
        if args.at:
            goal['at'] = args.at
        summary['goal'] = goal
        summary['origin'] = status['pos']
        summary['outcome'] = 'running'
        dispatched = call('task', 'attack', json.dumps(goal), '150', as_json=True)
        task_id = dispatched['id']
        summary['task'] = task_id
        print('Testing one native acquisition/kiting/melee goal', flush=True)
        anchor = datetime.datetime.now(datetime.timezone.utc).isoformat().replace('+00:00', 'Z')
        deadline = time.monotonic() + 155
        while time.monotonic() < deadline:
            status, mapped = observe()
            result = call('panel', 'GET', '/api/task?id=' + task_id + '&at=' + anchor, as_json=True)
            (saved / 'task.json').write_text(json.dumps(result, indent=2) + '\n')
            ended = terminal(result, task_id)
            if ended:
                summary.update(outcome=ended['status'], result=ended)
                break
            if (status.get('task') or {}).get('id') != task_id:
                raise RuntimeError('trial attack lost body ownership')
            combat = status.get('combat') or {}
            if combat.get('id') == task_id and combat.get('entity'):
                summary['entity'] = combat['entity']
            reason = abort_reason(status, summary['origin'], (mapped.get('perception') or {}).get('characters', []), summary.get('entity'))
            if reason:
                raise RuntimeError(reason)
            time.sleep(.5)
        else:
            raise RuntimeError('trial time limit')
    except Exception as e:
        summary.update(outcome='aborted', reason=str(e))
    finally:
        # Always save/logout and stop; a failed trial never requests owner clearance.
        for arguments in [('panel', 'POST', '/api/act/logout', '{}'), ('stop',), ('combat', 'false')]:
            try:
                call(*arguments, timeout=200)
            except Exception as e:
                summary.setdefault('cleanupErrors', []).append(str(e))
        until = datetime.datetime.now(datetime.timezone.utc).isoformat().replace('+00:00', 'Z')
        try:
            audit = call('audit-window', since, until, timeout=60).strip()
            summary['audit'] = audit
            if summary.get('task'):
                try:
                    summary['ground'] = ground_diagnostics(audit, summary['task'])
                except Exception as e:
                    summary['diagnosticError'] = str(e)
        except Exception as e:
            summary['auditError'] = str(e)
        (saved / 'summary.json').write_text(json.dumps(summary, indent=2) + '\n')
    report = {k: summary[k] for k in ('outcome', 'reason', 'entity', 'minimumHP', 'evidence', 'cleanupErrors') if k in summary}
    if summary.get('result'):
        report['result'] = {k: summary['result'][k] for k in ('status', 'reason', 'detail') if k in summary['result']}
    if summary.get('ground'):
        report['ground'] = {k: summary['ground'][k] for k in ('candidates', 'reasons', 'probeStages')}
    print(json.dumps(report, indent=2))
    return 0 if summary['outcome'] == 'succeeded' and not summary.get('cleanupErrors') else 1


if __name__ == '__main__':
    sys.exit(main())
