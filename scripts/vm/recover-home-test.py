#!/usr/bin/env python3
"""Exercise native-recovery supervision and cleanup without a live game."""
import contextlib
import importlib.util
import io
import json
from pathlib import Path
import tempfile
from types import SimpleNamespace
import unittest
from unittest.mock import patch, MagicMock

spec = importlib.util.spec_from_file_location('recover_home', Path(__file__).with_name('recover-home.py'))
recovery = importlib.util.module_from_spec(spec)
spec.loader.exec_module(recovery)

HEALTHY = dict(connected=True, phase='in_world', world='one', connection=2, hp=65, hpMax=68,
               pos=[0, 30, 0], flags=['in_shelter'], home=dict(world='one', pos=[0, 30, 0]), injurySeq=0)
OUTSIDE = HEALTHY | dict(hp=53, pos=[40, 30, 0], flags=['recovering'])


class RecoverySupervisorTests(unittest.TestCase):
    def run_trial(self, snapshots, *, reject_arm=False, unsafe_map=False):
        calls = []
        snapshots = iter(snapshots)

        def request(command, **kwargs):
            args = command[2:]
            calls.append(args)
            if args == ['panel', 'GET', '/api/status']:
                value = next(snapshots)
            elif args == ['panel', 'GET', '/api/map']:
                value = dict(world='one', connection=3 if unsafe_map else 2, perception=dict(characters=[]))
            elif args[:2] == ['recovery-return', 'arm']:
                if reject_arm:
                    return SimpleNamespace(returncode=1, stdout='', stderr='refused')
                value = dict(World='one', Pos=[0, 30, 0], Expires=1200)
            else:
                return SimpleNamespace(returncode=0, stdout='', stderr='')
            return SimpleNamespace(returncode=0, stdout=json.dumps(value), stderr='')

        launch = MagicMock()
        launch.poll.return_value = 0
        launch.returncode = 0
        with tempfile.TemporaryDirectory() as temp, patch.object(recovery, 'ROOT', Path(temp)), \
                patch.object(recovery.subprocess, 'run', side_effect=request), \
                patch.object(recovery.subprocess, 'Popen', return_value=launch), \
                patch.object(recovery.time, 'sleep'), patch('sys.argv', ['recover-home', 'one']), \
                contextlib.redirect_stdout(io.StringIO()):
            code = recovery.main()
            summary = json.loads(next(Path(temp).glob('.checks/recovery-*/summary.json')).read_text())
        return code, summary, calls

    def test_safe_native_return_can_remain_online_without_any_body_orders(self):
        code, result, calls = self.run_trial([dict(connected=False), OUTSIDE, HEALTHY])
        self.assertEqual(code, 0, result)
        self.assertNotIn(['stop'], calls)
        self.assertIn(['recovery-return', 'clear'], calls)
        self.assertFalse(any(c[0] in ('task', 'task-check') for c in calls))

    def test_injury_or_native_logout_or_wrong_scope_always_cleans_up(self):
        for last in (OUTSIDE | dict(hp=52, injurySeq=1), dict(connected=False),
                     HEALTHY | dict(connection=3), HEALTHY | dict(task=dict(id='owner'))):
            with self.subTest(last=last):
                code, result, calls = self.run_trial([dict(connected=False), OUTSIDE, last])
                self.assertEqual(code, 1, result)
                for action in (['stop'], ['panel-start'], ['recovery-return', 'clear']):
                    self.assertIn(action, calls)

    def test_refused_arm_does_not_stop_someone_elses_client(self):
        code, result, calls = self.run_trial([dict(connected=False)], reject_arm=True)
        self.assertEqual(code, 1, result)
        self.assertNotIn(['stop'], calls)

    def test_release_needs_same_scope_map_and_safe_home(self):
        code, result, calls = self.run_trial([dict(connected=False), OUTSIDE, HEALTHY], unsafe_map=True)
        self.assertEqual(code, 1, result)
        self.assertIn(['stop'], calls)


if __name__ == '__main__':
    unittest.main()
