"""Read retained harvest evidence without starting the VM or fetching broad history."""
import json
from pathlib import Path
import sys

from harvest_trial import diagnostics

if not 3 <= len(sys.argv) <= 7:
    raise SystemExit('usage: just vm harvest-evidence AUDIT TASK_ID [up to five IDs]')
audit = Path(sys.argv[1])
if not audit.is_absolute():
    audit = Path(__file__).resolve().parents[2] / audit
print(json.dumps(diagnostics(audit, sys.argv[2:]), indent=2))
