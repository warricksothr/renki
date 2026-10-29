#!/usr/bin/env python3
"""Send one VM task, retain its evidence, and print one terminal result."""
import datetime
import json
import math
from pathlib import Path
import subprocess
import sys
import tempfile
import time


def main(task, args, timeout):
    seconds = float(timeout)
    if not math.isfinite(seconds) or seconds <= 0:
        raise SystemExit("task-check requires a finite positive timeout")
    root = Path(__file__).resolve().parents[2]
    saved = root / ".checks"
    saved.mkdir(exist_ok=True)
    with tempfile.NamedTemporaryFile(prefix="task-", suffix=".json", dir=saved, delete=False) as handle:
        path = Path(handle.name)
    log = path.with_suffix(".log")

    def call(*arguments):
        result = subprocess.run(["just", "vm", *arguments], cwd=root, capture_output=True, text=True, timeout=45)
        with log.open("a") as out:
            out.write(result.stdout + result.stderr)
        if result.returncode:
            raise SystemExit(f"VM command failed; retained diagnostics: {log}")
        return json.loads(result.stdout)

    dispatched = call("task", task, args, timeout)
    task_id = dispatched["id"]
    anchor = datetime.datetime.now(datetime.timezone.utc).isoformat().replace("+00:00", "Z")
    deadline = time.monotonic() + seconds + 15
    while time.monotonic() < deadline:
        result = call("panel", "GET", f"/api/task?id={task_id}&at={anchor}")
        path.write_text(json.dumps(result, indent=2) + "\n")
        records = result.get("value", []) if isinstance(result, dict) else result
        for record in reversed(records):
            frame = record.get("frame", {})
            if frame.get("type") != "task" or frame.get("re") != task_id:
                continue
            status = frame.get("status")
            if status in ("succeeded", "failed", "rejected", "cancelled"):
                detail = " ".join(str(frame.get(key, "")) for key in ("reason", "detail")).strip()
                if len(detail) > 600: detail = detail[:600] + "…"
                print(f"{task_id} {task}: {status}" + (f" ({detail})" if detail else ""))
                print(f"Evidence: {path}")
                return 0 if status == "succeeded" else 1
        time.sleep(2)
    raise SystemExit(f"{task_id} {task}: no terminal result within the wait limit; evidence: {path}; diagnostics: {log}")


if __name__ == "__main__":
    if len(sys.argv) != 4:
        raise SystemExit("usage: task-check.py TASK ARGS TIMEOUT")
    sys.exit(main(*sys.argv[1:]))
