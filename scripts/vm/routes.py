#!/usr/bin/env python3
"""Print /api/routes as a short report: `just vm routes`.

Route memory is the ground the body has walked (docs/ROUTES.md). This is the
first thing to read when a move fails or a plan cannot be made: it says how
much is known, on which height levels, whether the known ground is in one
piece, and whether the body stands on it.
"""
import json
import sys


def render(status):
    stats = status.get('stats') or {}
    if not status.get('world'):
        return 'routes: the bot is not in a world'
    lines = [f"routes in {status['world']}: {stats.get('cells', 0)} walked cells, {stats.get('links', 0)} links "
             f"({stats.get('walked', 0)} walked, {stats.get('links', 0) - stats.get('walked', 0)} inferred back, "
             f"{stats.get('failed', 0)} failed)"]
    if stats.get('newest'):
        lines.append(f"  newest recording: {stats['newest'][:19]}Z")
    levels = stats.get('levels') or []
    if levels:
        shown = ', '.join(f"{level['y']:g} m ({level['cells']})" for level in levels[:16])
        lines.append(f"  height levels: {shown}" + (f", and {len(levels) - 16} more" if len(levels) > 16 else ''))
    groups = stats.get('groups') or []
    if len(groups) > 1:
        lines.append(f"  {len(groups)} separate groups: no walked way between them")
    for group in groups[:8]:
        x, y, z = group['centre']
        lines.append(f"    {group['cells']:>5} cells around [{x:.0f}, {y:.0f}, {z:.0f}], "
                     f"heights {group['minY']:.1f} to {group['maxY']:.1f} m")
    if len(groups) > 8:
        lines.append(f"    and {len(groups) - 8} smaller groups")
    body = status.get('body')
    if body:
        where = ', '.join(f'{n:.1f}' for n in body['pos'])
        if body.get('known'):
            lines.append(f"  body at [{where}] stands on walked ground")
        elif body.get('nearest'):
            near = ', '.join(f'{n:.1f}' for n in body['nearest'])
            lines.append(f"  body at [{where}] is off walked ground; nearest is [{near}], {body['nearestM']:.1f} m away")
        else:
            lines.append(f"  body at [{where}]; nothing walked yet")
    return '\n'.join(lines)


if __name__ == '__main__':
    raw = sys.stdin.read()
    print(render(json.loads(raw[raw.index('{'):])))
