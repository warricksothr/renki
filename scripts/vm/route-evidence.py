#!/usr/bin/env python3
"""Bounded projection of retained route/support evidence; never query the game."""
import argparse
from collections import Counter, deque
import json
from pathlib import Path


def query(value):
    if not value:
        return None
    return {k: (v[:3] if k == 'contacts' else v) for k, v in value.items()}


def failures(value):
    if not value:
        return None
    examples = [{k: (query(v) if k == 'supportQuery' else v) for k, v in sample.items()}
                for sample in value.get('examples', [])[:3]]
    return dict(counts=value.get('counts', {}), examples=examples)


def project(audit, limit=12):
    counts, records = Counter(), deque(maxlen=limit)
    for line in audit:
        try:
            row = json.loads(line)
        except ValueError:
            continue
        frame = row.get('frame') or {}
        if frame.get('type') != 'trace' or frame.get('source') not in ('walker', 'navigation'):
            continue
        data = frame.get('data') or {}
        if not any(k in data for k in ('supportQuery', 'planning', 'failures')):
            continue
        counts[frame.get('msg', '')] += 1
        record = dict(at=row.get('at'), task=frame.get('task'), message=frame.get('msg'))
        record.update({k: data[k] for k in ('pos', 'from', 'reason', 'failure', 'checking', 'checkingFrom',
                       'searchStatus', 'expanded', 'probes', 'elapsedS', 'supportQuery', 'failures', 'planning',
                       'stagingLegs', 'progressLeg', 'end', 'piece', 'frontier') if k in data})
        if 'supportQuery' in record: record['supportQuery'] = query(record['supportQuery'])
        if 'failures' in record: record['failures'] = failures(record['failures'])
        if record.get('planning'):
            record['planning'] = {stage: (dict(state, failures=failures(state.get('failures'))) if state else None)
                                  for stage, state in record['planning'].items()}
        records.append(record)
    return dict(events=dict(counts), recent=list(records))


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('audit', type=Path)
    parser.add_argument('--limit', type=int, choices=range(1, 25), default=6)
    args = parser.parse_args()
    if not args.audit.is_absolute():
        args.audit = Path(__file__).resolve().parents[2] / args.audit
    with args.audit.open(encoding='utf-8-sig') as stream:
        result = project(stream, args.limit)
    print(json.dumps(result, indent=2))


if __name__ == '__main__':
    main()
