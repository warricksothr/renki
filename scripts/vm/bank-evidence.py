#!/usr/bin/env python3
"""Summarize a retained bank-refusal sweep without game requests or full frames."""
import argparse
import json
from pathlib import Path


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('audit', type=Path, help='saved audit-window JSONL')
    args = parser.parse_args()
    if not args.audit.is_absolute():
        args.audit = Path(__file__).resolve().parents[2] / args.audit
    latest = None
    with args.audit.open(encoding='utf-8-sig') as rows:
        for line in rows:
            if not line.strip():
                continue
            frame = json.loads(line).get('frame', {})
            if frame.get('type') == 'trace' and frame.get('source') == 'walker' and frame.get('msg') == 'dry bank departure refused':
                latest = frame.get('data') or {}
    if latest is None:
        raise SystemExit('No retained dry-bank refusal in this audit window')
    failures = latest.get('failures') or {}
    print('Evidence:', args.audit)
    print('Body:', latest.get('pos'), 'water:', latest.get('waterLevel'),
          'eligible:', latest.get('eligible'), 'samples:', latest.get('samples'))
    print('Reason:', latest.get('reason', 'not recorded by this build'))
    print('Probe:', json.dumps(failures.get('bank'), separators=(',', ':')))
    print('Refusals:', json.dumps(failures.get('counts', {}), separators=(',', ':')))
    for i, sample in enumerate(failures.get('examples', [])[:16], 1):
        heights, water = sample.get('supportHeights') or [], sample.get('waterLevels') or []
        result = dict(candidate=i, stage=sample.get('stage'), at=sample.get('at'),
                      support=[round(v, 4) for v in heights], water=[round(v, 4) for v in water],
                      stepDistance=sample.get('stepDistance'), bankClear=sample.get('bankClear'))
        if len(heights) == 4 and len(water) == 3:
            body, foot, middle, end = heights
            offset = max(0, foot - body) + .1
            result['rise'] = [round(middle-foot, 4), round(end-middle, 4)]
            result['settledClearance'] = [round(min(body, foot)-.1-water[0], 4),
                                          round(middle-offset-water[1], 4), round(end-offset-water[2], 4)]
        print(json.dumps(result, separators=(',', ':')))


if __name__ == '__main__':
    main()
