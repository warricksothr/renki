#!/usr/bin/env python3
"""Exercise native-recovery supervision and cleanup without a live game."""
import contextlib
import importlib.util
import io
import itertools
import json
import os
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
    def run_trial(self, snapshots, *, reject_arm=False, unsafe_map=False, work_trip=False, nearby=False, capture=False,
                  audit_frames=None, join_failed=False):
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
            elif args[0] == 'audit-window' and audit_frames is not None:
                audit = Path(temp) / 'window.jsonl'
                audit.write_text(''.join(json.dumps(row) + '\n' for row in audit_frames))
                return SimpleNamespace(returncode=0, stdout=str(audit), stderr='')
            else:
                return SimpleNamespace(returncode=0, stdout='', stderr='')
            return SimpleNamespace(returncode=0, stdout=json.dumps(value), stderr='')

        launch = MagicMock()
        launch.poll.return_value = 0
        launch.returncode = 1 if join_failed else 0
        clock = itertools.count(0, 3)
        with tempfile.TemporaryDirectory() as temp, patch.object(recovery, 'ROOT', Path(temp)), \
                patch.object(recovery.subprocess, 'run', side_effect=request), \
                patch.object(recovery.subprocess, 'Popen', return_value=launch), \
                patch.object(recovery.time, 'monotonic', side_effect=lambda: next(clock)), \
                patch.object(recovery.time, 'sleep'), patch('sys.argv', ['recover-home', 'one'] + (['--work-trip'] if work_trip else ['--nearby'] if nearby else []) + (['--capture'] if capture else [])), \
                contextlib.redirect_stdout(io.StringIO()):
            code = recovery.main()
            summary = json.loads(next(Path(temp).glob('.checks/recovery-*/summary.json')).read_text())
        return code, summary, calls

    def test_native_refusal_between_polls_explains_join_failure_without_starting_work(self):
        frame = dict(at='2026-11-01T10:37:11Z', conn=1,
                     frame=dict(type='trace', source='recovery', seq=119, msg='startup return refused',
                                data=dict(reason='unsafe health, food, water or threat clearance',
                                          hp=66.9551, hpMax=66.9551, foods=2, threatDistance=.64487)))
        code, result, calls = self.run_trial([dict(connected=False), dict(connected=True, phase='menu')],
                                              audit_frames=[frame], join_failed=True)
        self.assertEqual(code, 1)
        self.assertEqual(result['supervisorReason'], 'join failed')
        self.assertEqual(result['reason'], 'native recovery refused: unsafe health, food, water or threat clearance')
        self.assertEqual(result['nativeRecovery']['refusal']['threatDistance'], .64487)
        self.assertEqual(result['nativeRecovery']['refusal']['connection'], 1)
        self.assertEqual(result['nativeRecovery']['refusal']['seq'], 119)
        self.assertFalse(any(c[0] == 'task' for c in calls))
        for action in (['stop'], ['panel-start'], ['recovery-return', 'clear']):
            self.assertIn(action, calls)

    def test_native_diagnostics_do_not_infer_success_or_hide_other_failures(self):
        with tempfile.TemporaryDirectory(dir=recovery.ROOT / '.checks') as temp:
            path = Path(temp) / 'window.jsonl'
            refusal = dict(conn=4, frame=dict(type='trace', source='recovery', seq=2,
                                             msg='startup return refused', data=dict(reason='unsafe')))
            path.write_text('\ufeff' + json.dumps(dict(frame=dict(type='trace', source='other',
                                                                 msg='startup return refused', data=dict(reason='wrong')))) + '\n'
                            + 'invalid row\n' + json.dumps(refusal) + '\n')
            for outcome, reason in [('passed', 'native return and sheltered recovery observed'),
                                    ('failed', 'injury outside owned food travel budget')]:
                result = dict(outcome=outcome, reason=reason)
                recovery.explain_native_refusal(result, path)
                self.assertEqual(result['outcome'], outcome)
                self.assertEqual(result['reason'], reason)
                self.assertEqual(result['nativeRecovery']['refusal']['reason'], 'unsafe')
                self.assertEqual(result['nativeRecovery']['malformedRows'], 1)
                self.assertNotIn('supervisorReason', result)
            report = recovery.subprocess.run(['just', 'vm', 'recovery-evidence', str(path.relative_to(recovery.ROOT))], cwd=recovery.ROOT,
                                             capture_output=True, text=True, timeout=10)
            self.assertEqual(report.returncode, 0, report.stderr)
            self.assertEqual(json.loads(report.stdout)['nativeRecovery']['refusal']['connection'], 4)

    def test_safe_native_return_can_remain_online_without_any_body_orders(self):
        code, result, calls = self.run_trial([dict(connected=False), OUTSIDE, HEALTHY])
        self.assertEqual(code, 0, result)
        self.assertNotIn(['stop'], calls)
        self.assertIn(['recovery-return', 'clear'], calls)
        self.assertFalse(any(c[0] in ('task', 'task-check') for c in calls))

    def test_explicit_work_trip_profile_is_forwarded_without_body_orders(self):
        code, result, calls = self.run_trial([dict(connected=False), OUTSIDE, HEALTHY], work_trip=True)
        self.assertEqual(code, 0, result)
        self.assertEqual(result['profile'], 'work_trip')
        self.assertIn(['recovery-return', 'arm', 'one', 'work_trip'], calls)
        self.assertIn(['recovery-return', 'clear'], calls)
        self.assertFalse(any(c[0] in ('task', 'task-check') for c in calls))

    def test_optional_camera_checkpoint_is_read_once_without_config_or_body_changes(self):
        code, result, calls = self.run_trial([dict(connected=False), OUTSIDE, OUTSIDE, HEALTHY], capture=True)
        self.assertEqual(code, 0, result)
        self.assertEqual(calls.count(['camera']), 1)
        self.assertFalse(any(c[0] in ('camera-rate', 'camera-config', 'task') for c in calls))

    def test_just_launcher_accepts_nearby_without_contacting_a_vm(self):
        with tempfile.TemporaryDirectory() as temp:
            bindir = Path(temp)
            for name, body in [('ssh', 'printf "%s\\n" "$*"'), ('scp', 'exit 0')]:
                target = bindir / name
                target.write_text('#!/bin/sh\n' + body + '\n')
                target.chmod(0o755)
            env = os.environ | dict(RENKI_VM='unittest@invalid', PATH=str(bindir)+os.pathsep+os.environ['PATH'])
            result = recovery.subprocess.run(['just', 'vm', 'recovery-return', 'arm', 'one', 'nearby'], cwd=recovery.ROOT,
                                            env=env, capture_output=True, text=True, timeout=10)
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertIn("-ReturnProfile 'nearby'", result.stdout)

    def test_explicit_nearby_profile_is_forwarded_without_body_orders(self):
        code, result, calls = self.run_trial([dict(connected=False), OUTSIDE | dict(hp=44, hpMax=53.5, pos=[20,30,0]), HEALTHY], nearby=True)
        self.assertEqual(code, 0, result)
        self.assertEqual(result['profile'], 'nearby')
        self.assertIn(['recovery-return', 'arm', 'one', 'nearby'], calls)
        self.assertFalse(any(c[0] in ('task', 'task-check') for c in calls))

    def test_work_trip_injury_still_stops_and_clears_the_order(self):
        code, result, calls = self.run_trial([dict(connected=False), OUTSIDE, OUTSIDE | dict(hp=52, injurySeq=1)], work_trip=True)
        self.assertEqual(code, 1, result)
        for action in (['stop'], ['panel-start'], ['recovery-return', 'clear']):
            self.assertIn(action, calls)

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
