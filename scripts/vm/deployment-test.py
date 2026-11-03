#!/usr/bin/env python3
"""Check safety ordering in the entry points whose compatibility policy is Go-tested."""
from pathlib import Path
import os
import subprocess
import unittest

ROOT = Path(__file__).resolve().parents[2]


class DeploymentEntrypoints(unittest.TestCase):
    def test_full_install_stages_and_checks_before_stop(self):
        result = subprocess.run(['just', '--dry-run', 'vm', 'install'], cwd=ROOT,
                                capture_output=True, text=True, check=True,
                                env={**os.environ, 'RENKI_VM': 'fake@offline'})
        plan = result.stderr + result.stdout
        self.assertLess(plan.index("-File '$stage/deployment.ps1'"), plan.index('vm stop'))
        self.assertLess(plan.index("-File '$stage/deployment.ps1'"), plan.index('vm panel-stop'))
        self.assertNotIn('vm start', plan)
        self.assertIn(':/\u0024stage/', plan)

    def test_both_native_installers_refuse_before_stopping_or_copying(self):
        for name, mutation in [('install.ps1', 'Copy-Item'), ('deploy-controller.ps1', 'Stop-Process')]:
            source = (ROOT / 'scripts/vm' / name).read_text()
            self.assertLess(source.index('Test-Deployment $s $Root $Stage'), source.index(mutation))
        controller = (ROOT / 'scripts/vm/deploy-controller.ps1').read_text()
        self.assertNotIn('Start-ScheduledTask -TaskName RenkiStart', controller)
        self.assertIn('if ($controllers.Count -gt 0)', controller)


if __name__ == '__main__':
    unittest.main()
