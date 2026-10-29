#!/usr/bin/env python3
"""Bounded evidence from a saved audit window; keep full frames on disk."""
import collections
import json
import sys


def main(path):
    tasks = collections.deque(maxlen=20)
    traces = collections.deque(maxlen=10)
    containers = {}
    inventory = None
    first = last = None
    minimum_hp = None
    counts = collections.Counter()
    for number, line in enumerate(open(path, encoding="utf-8-sig"), 1):
        if not line.strip():
            continue
        try:
            record = json.loads(line)
        except ValueError as error:
            raise SystemExit(f"{path}:{number}: {error}")
        frame = record.get("frame", {})
        kind = frame.get("type", record.get("kind")) or "unknown"
        counts[kind] += 1
        if kind == "state" and frame.get("phase") == "in_world":
            first = first or frame
            last = frame
            hp = frame.get("hp")
            if hp is not None:
                minimum_hp = hp if minimum_hp is None else min(minimum_hp, hp)
        elif kind == "task":
            text = " ".join(str(frame[key]) for key in ("id", "re", "task", "status", "reason", "detail") if key in frame)
            tasks.append(f"{record['at']} {text}")
        elif kind == "trace" and frame.get("source") in ("walker", "harvest", "transfer"):
            counts["movement/harvest traces"] += 1
            data = json.dumps(frame.get('data', {}), separators=(',', ':'))
            if len(data) > 400:
                data = data[:400] + "…"
            traces.append(f"{record['at']} {frame.get('task', '')} {frame.get('msg', '')} {data}")
        elif kind == "inventory":
            inventory = frame
        elif kind == "container":
            containers[frame.get("entity")] = frame
    print(f"Full evidence: {path}")
    print("Frames: " + ", ".join(f"{key}={value}" for key, value in sorted(counts.items()) if key))
    if last:
        print(f"State: {first.get('pos')} -> {last.get('pos')}; minimum HP={minimum_hp}; last task={last.get('task')}")
    for task in tasks:
        print(task)
    for trace in traces:
        print(trace)
    if inventory:
        print("Last pack: " + items(inventory))
    for entity, frame in list(containers.items())[-6:]:
        print(f"Last chest {entity}: {items(frame)}; moved={frame.get('moved', {})}; remaining={frame.get('remaining', {})}")


def items(frame):
    totals = collections.Counter()
    for item in frame.get("items", []):
        totals[item.get("prefab", "?")] += item.get("stack", 0)
    return ", ".join(f"{name}={count}" for name, count in sorted(totals.items())) or "empty"


if __name__ == "__main__":
    main(sys.argv[1])
