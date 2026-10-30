"""Supervisor stays with one objective through failed native attempts."""
import contextlib
import importlib.util
import io
import json
from pathlib import Path
import tempfile
import unittest
from unittest import mock

spec = importlib.util.spec_from_file_location('trial', Path(__file__).with_name('combat-trial.py'))
trial = importlib.util.module_from_spec(spec)
spec.loader.exec_module(trial)


class Goals(unittest.TestCase):
    def test_child_failure_does_not_end_objective_and_proof_is_required(self):
        for ending in ('succeeded', 'needs_plan', 'missing_proof'):
            with self.subTest(ending=ending), tempfile.TemporaryDirectory() as folder:
                base = dict(id='goal-1', status='running', attempts=1, reason='')
                failed = dict(base, reason='travel limit', last=dict(status='failed'))
                final = dict(base, status='needs_plan' if ending == 'needs_plan' else 'succeeded', attempts=2)
                if ending == 'succeeded':
                    final['proof'] = dict(taskId='native-2', detail='observed Greyling death')
                frames = iter([None, base, failed, final])
                status = dict(connected=True, phase='in_world', world='one', pos=[0, 30, 0], hp=80, hpMax=80, stamina=60)
                def observe():
                    goal = next(frames)
                    return dict(status, goals=[] if goal is None else [goal]), {}
                def call(*args, **kwargs):
                    self.assertEqual(args[:3], ('panel', 'POST', '/api/goals'))
                    self.assertEqual(json.loads(args[3])['spec']['world'], 'one')
                    return dict(id='goal-1')
                summary = {}
                with mock.patch.object(trial.time, 'sleep'), contextlib.redirect_stdout(io.StringIO()):
                    if ending == 'missing_proof':
                        with self.assertRaisesRegex(RuntimeError, 'lacks host proof'):
                            trial.run_persistent_goal(summary, call, observe, Path(folder))
                    else:
                        trial.run_persistent_goal(summary, call, observe, Path(folder))
                        self.assertEqual(summary['outcome'], ending)
                        self.assertEqual(summary['goal']['attempts'], 2)
                self.assertEqual(summary['goalId'], 'goal-1')

    def test_cleanup_preserves_needs_plan_but_cancels_an_aborted_goal(self):
        for ending in ('needs_plan', 'aborted'):
            with self.subTest(ending=ending), tempfile.TemporaryDirectory() as folder:
                commands, observed = [], 0
                def run(command, **kwargs):
                    nonlocal observed
                    args = command[2:]
                    commands.append(args)
                    value = ''
                    if args[:3] == ['panel', 'GET', '/api/status']:
                        observed += 1
                        value = dict(connected=True, phase='in_world', world='one', pos=[0, 30, 0], hp=80, hpMax=80, stamina=60, food=dict(active=[dict(prefab='Raspberry')]))
                        if observed >= 3:
                            value['goals'] = [dict(id='goal-1', status='needs_plan' if ending == 'needs_plan' else 'running', attempts=1, reason='no changed target')]
                            if ending == 'aborted':
                                value['hp'] = 0
                    elif args[:3] == ['panel', 'GET', '/api/map']:
                        value = dict(perception=dict(characters=[]))
                    elif args[:3] == ['panel', 'POST', '/api/goals']:
                        value = dict(id='goal-1')
                    return mock.Mock(returncode=0, stdout=value if isinstance(value, str) else json.dumps(value), stderr='')
                with mock.patch.object(trial, 'ROOT', Path(folder)), mock.patch.object(trial.subprocess, 'run', side_effect=run), mock.patch.object(trial.sys, 'argv', ['trial', '--persistent', '--wait-s', '0']), contextlib.redirect_stdout(io.StringIO()):
                    self.assertEqual(trial.main(), 1)
                stop_goals = [json.loads(c[3]) for c in commands if c[:3] == ['panel', 'POST', '/api/goals'] and json.loads(c[3])['action'] == 'stop']
                self.assertEqual(len(stop_goals), 1 if ending == 'aborted' else 0)
                self.assertIn(['combat', 'false'], commands)
                self.assertEqual(commands[-2], ['panel-start'])

    def test_persistent_mode_disallows_conflicting_modes(self):
        for arguments in (['--direct', '--entity', 'greyling'], ['--entity', 'greyling'], ['--harvest-tree'], ['--at', '0', '0', '0']):
            with self.subTest(arguments=arguments), mock.patch.object(trial.sys, 'argv', ['trial', '--persistent', *arguments]), mock.patch.object(trial.subprocess, 'run') as run, contextlib.redirect_stderr(io.StringIO()):
                with self.assertRaises(SystemExit):
                    trial.main()
                run.assert_not_called()


if __name__ == '__main__':
    unittest.main()
