#!/usr/bin/env python3
"""Read native recovery refusals from one retained audit; never contact the VM."""
import argparse
import importlib.util
import json
from pathlib import Path

spec = importlib.util.spec_from_file_location('recover_home', Path(__file__).with_name('recover-home.py'))
recovery = importlib.util.module_from_spec(spec)
spec.loader.exec_module(recovery)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('audit', type=Path)
    args = parser.parse_args()
    if not args.audit.is_absolute():
        args.audit = recovery.ROOT / args.audit
    print(json.dumps(dict(audit=str(args.audit), nativeRecovery=recovery.recovery_diagnostics(args.audit)), indent=2))


if __name__ == '__main__':
    main()
