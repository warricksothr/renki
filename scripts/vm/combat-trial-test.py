import importlib.util
import contextlib
import io
import json
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest import mock

spec = importlib.util.spec_from_file_location('trial', Path(__file__).with_name('combat-trial.py'))
trial = importlib.util.module_from_spec(spec)
spec.loader.exec_module(trial)


class CombatTrialTests(unittest.TestCase):
    def setUp(self):
        self.status = dict(connected=True, phase='in_world', hp=80, hpMax=80, pos=[0, 32, 0], stamina=60)
        self.enemy = dict(id='greyling', kind='creature', prefab='Greyling', pos=[25, 32, 0], hpFrac=1, hostile=True)
        self.mapped = dict(perception=dict(characters=[self.enemy], structures=[]))

    def test_only_eligible_single_target_in_clear_area(self):
        self.assertEqual(trial.candidates(self.status, self.mapped), [self.enemy])
        for field, value in [('stars', 1), ('tamed', True), ('hpFrac', 0), ('hostile', False), ('prefab', 'Boar'), ('kind', 'player'), ('pos', [31, 32, 0])]:
            with self.subTest(field=field):
                enemy = dict(self.enemy, **{field: value})
                self.assertEqual(trial.candidates(self.status, dict(perception=dict(characters=[enemy]))), [])

    def test_bystanders_and_structures_at_either_end_prevent_trial(self):
        for pos in ([1, 32, 0], [24, 32, 0]):
            with self.subTest(pos=pos):
                bystander = dict(id='owner', kind='player', hpFrac=1, pos=pos)
                self.mapped['perception']['characters'] = [self.enemy, bystander]
                self.assertEqual(trial.candidates(self.status, self.mapped), [])
                self.mapped['perception']['characters'] = [self.enemy]
                self.mapped['perception']['structures'] = [dict(pos=pos)]
                self.assertEqual(trial.candidates(self.status, self.mapped), [])
                self.mapped['perception']['structures'] = []

    def test_low_health_exhausted_escape_and_long_chases_abort(self):
        self.assertIsNone(trial.abort_reason(self.status, [0, 32, 0]))
        self.assertEqual(trial.abort_reason(dict(self.status, hp=55)), 'health below trial floor')
        self.assertEqual(trial.abort_reason(dict(self.status, combat=dict(mode='retreat'), stamina=0)), 'escape stamina exhausted')
        self.assertEqual(trial.abort_reason(dict(self.status, pos=[41, 32, 0]), [0, 32, 0]), 'trial travel limit')
        boar = dict(self.enemy, id='boar', prefab='Boar', pos=[7, 32, 0])
        self.assertEqual(trial.abort_reason(self.status, characters=[boar], target='greyling'), 'additional nearby creature')
        self.assertEqual(trial.abort_reason(dict(self.status, phase='menu')), 'not in the world')

    def test_only_matching_task_terminal_status_ends_the_trial(self):
        result = [dict(frame=dict(re='old', status='succeeded')), dict(frame=dict(re='trial', status='running'))]
        self.assertIsNone(trial.terminal(result, 'trial'))
        result.append(dict(frame=dict(re='trial', status='rejected', reason='refused')))
        self.assertEqual(trial.terminal(dict(value=result), 'trial')['status'], 'rejected')

    def test_runner_stops_and_restores_gate_after_success_or_additional_attacker(self):
        for outcome in ('success', 'extra attacker'):
            with self.subTest(outcome=outcome), tempfile.TemporaryDirectory() as directory:
                commands = []
                status = dict(self.status, food=dict(active=[dict(prefab='CookedDeerMeat')]))
                observed = 0

                def run(command, **kwargs):
                    nonlocal observed
                    args = command[2:]
                    commands.append(args)
                    value = ''
                    if args[:3] == ['panel', 'GET', '/api/status']:
                        observed += 1
                        value = dict(status, **({'task': dict(id='trial', task='attack')} if observed > 1 else {}))
                    elif args[:3] == ['panel', 'GET', '/api/map']:
                        chars = [self.enemy]
                        if observed > 1 and outcome == 'extra attacker':
                            chars.append(dict(self.enemy, id='boar', prefab='Boar', pos=[7, 32, 0]))
                        value = dict(perception=dict(characters=chars, structures=[]))
                    elif args[0] == 'task':
                        value = dict(id='trial')
                    elif args[:2] == ['panel', 'GET'] and args[2].startswith('/api/task?'):
                        value = [dict(frame=dict(re='trial', status='succeeded'))] if outcome == 'success' else []
                    elif args[0] == 'audit-window':
                        value = '/saved/audit.jsonl'
                    return subprocess.CompletedProcess(command, 0, json.dumps(value) if isinstance(value, (dict, list)) else value, '')

                with mock.patch.object(trial, 'ROOT', Path(directory)), mock.patch.object(trial.subprocess, 'run', side_effect=run), mock.patch.object(trial.sys, 'argv', ['combat-trial', '--wait-s', '0']), contextlib.redirect_stdout(io.StringIO()):
                    code = trial.main()
                self.assertEqual(code, 0 if outcome == 'success' else 1)
                self.assertEqual(commands.count(['stop']), 2)
                self.assertIn(['combat', 'true'], commands)
                self.assertIn(['combat', 'false'], commands)
                self.assertIn(['panel', 'POST', '/api/act/logout', '{}'], commands)
                summary = json.loads(next(Path(directory).glob('.checks/*/summary.json')).read_text())
                self.assertEqual(summary['outcome'], 'succeeded' if outcome == 'success' else 'aborted')
                if outcome != 'success':
                    self.assertEqual(summary['reason'], 'additional nearby creature')


if __name__ == '__main__':
    unittest.main()
