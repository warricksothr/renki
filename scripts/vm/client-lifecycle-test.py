#!/usr/bin/env python3
"""Exercise real supervisor failure cleanup without a game or SSH."""
import contextlib
import importlib.util
import io
import json
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest.mock import patch


class ClientLifecycle(unittest.TestCase):
    def test_injured_cooking_food_and_camera_trials_require_explicit_shutdown(self):
        for name in ('cooking-check', 'food-check', 'camera-check'):
            for allow in (False, True):
                with self.subTest(tool=name, allow=allow), tempfile.TemporaryDirectory() as directory:
                    spec = importlib.util.spec_from_file_location(name, Path(__file__).with_name(name + '.py'))
                    module = importlib.util.module_from_spec(spec)
                    spec.loader.exec_module(module)
                    commands = []
                    state = dict(connected=True, phase='in_world', world='one', worldUid='uid', connection=1,
                                 hp=1, hpMax=100, pos=[0, 30, 0], flags=['grounded'])

                    def request(command, **kwargs):
                        args = command[2:]
                        commands.append(args)
                        value = {}
                        if args == ['panel', 'GET', '/api/status']:
                            value = state
                        elif args == ['panel', 'GET', '/api/camera']:
                            value = dict(config=dict(enabled=True, intervalS=5), baseConfig=dict(enabled=True, intervalS=5))
                        elif args == ['panel', 'GET', '/api/settings']:
                            value = []
                        return subprocess.CompletedProcess(command, 0, json.dumps(value), '')

                    with patch.object(module, 'ROOT', Path(directory)), \
                            patch.object(module.subprocess, 'run', side_effect=request), \
                            patch('sys.argv', [name] + (['--allow-logout'] if allow else [])), \
                            contextlib.redirect_stdout(io.StringIO()):
                        code = module.main()
                    self.assertEqual(code, 1)
                    self.assertEqual(['stop'] in commands, allow)
                    if name != 'camera-check':
                        self.assertEqual(['panel', 'POST', '/api/act/logout', '{}'] in commands, allow)
                    self.assertFalse(any(c[0] == 'combat' for c in commands))
                    evidence = list((Path(directory) / '.checks').rglob('*.json'))
                    self.assertTrue(evidence, 'failure evidence was not retained')


if __name__ == '__main__':
    unittest.main()
