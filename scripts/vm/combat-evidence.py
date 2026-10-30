"""Read a trial audit without starting the game; print bounded ground evidence."""
import importlib.util
import json
from pathlib import Path
import sys

if len(sys.argv) != 3:
    raise SystemExit('usage: just vm combat-evidence AUDIT TASK_ID')
spec = importlib.util.spec_from_file_location('combat_trial', Path(__file__).with_name('combat-trial.py'))
trial = importlib.util.module_from_spec(spec)
spec.loader.exec_module(trial)
audit = Path(sys.argv[1])
if not audit.is_absolute():
    audit = trial.ROOT / audit
summary = trial.ground_diagnostics(audit, sys.argv[2])
print(json.dumps({k: v for k, v in summary.items() if k != 'examples'}))
for example in summary['examples']:
    print(json.dumps(example))
