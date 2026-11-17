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


def plan_path(target, origin=''):
    """The API path for a plan. A target of three numbers is a position;
    anything else is a place name ("home", or a saved place)."""
    import re
    from urllib.parse import quote
    number = r'-?\d+(?:\.\d+)?'
    position = re.compile(rf'^{number},{number},{number}$')
    target = target.replace(' ', '') if position.match(target.replace(' ', '')) else target.strip()
    key = 'to' if position.match(target) else 'place'
    if not target:
        raise SystemExit('route-plan needs a target: x,y,z or a place name')
    path = f'/api/routes/plan?{key}={quote(target, safe="")}'
    if origin:
        origin = origin.replace(' ', '')
        if not position.match(origin):
            raise SystemExit('the start must be x,y,z')
        path += f'&from={quote(origin, safe="")}'
    return path


def spot(pos):
    return '[' + ', '.join(f'{n:.1f}' for n in pos) + ']'


def render_plan(plan):
    target = f"{plan['place']} {spot(plan['target']['pos'])}" if plan.get('place') else spot(plan['target']['pos'])
    lines = [f"route from {spot(plan['start']['pos'])} to {target}"]
    if not plan.get('found'):
        lines.append(f"  NO ROUTE ({plan.get('reason')}): {plan.get('explain')}")
        for name in ('start', 'target'):
            anchor = plan[name]
            if not anchor.get('cell') and anchor.get('cellPos'):
                lines.append(f"  nearest walked cell to the {name}: {spot(anchor['cellPos'])}, "
                             f"{anchor['distanceM']:.1f} m away")
        if plan.get('closest'):
            lines.append(f"  walk from {spot(plan['closest'])} toward the target to close the "
                         f"{plan['gapM']:.1f} m gap")
        return '\n'.join(lines)
    lines.append(f"  {plan['explain']}")
    for index, leg in enumerate(plan['legs'], 1):
        notes = []
        if abs(leg.get('climb', 0)) >= 0.4:
            notes.append(f"{'up' if leg['climb'] > 0 else 'down'} {abs(leg['climb']):.1f} m")
        if leg.get('inferred'):
            notes.append(f"{leg['inferred']} of {leg['cells']} links never walked this way")
        if not leg.get('cells'):
            notes.append('off walked ground')
        lines.append(f"  {index:>2}. {spot(leg['to'])}  {leg['lengthM']:.1f} m"
                     + (f"  ({'; '.join(notes)})" if notes else ''))
    return '\n'.join(lines)


if __name__ == '__main__':
    if len(sys.argv) > 1 and sys.argv[1] == 'plan-path':
        print(plan_path(*sys.argv[2:4]))
        sys.exit(0)
    raw = sys.stdin.read()
    if '{' not in raw:
        raise SystemExit(raw.strip() or 'no reply')
    data = json.loads(raw[raw.index('{'):])
    print(render_plan(data) if 'legs' in data else render(data))
