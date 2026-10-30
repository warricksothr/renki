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

    def test_ground_summary_filters_tasks_counts_failures_and_bounds_examples(self):
        def row(task, reason, stage=None):
            data = dict(candidate=[1, 32, 2], reason=reason, region='stance', segment=3, angle=34, previousAngle=22, roughness=.2)
            if stage:
                data.update(probeStage=stage, collider='Beech1/stem', probeAt=[2, 32, 2])
            return dict(frame=dict(type='trace', task=task, source='combat', msg='ground_rejected', data=data))
        rows = [row('old', 'ignored'), row('trial', 'grade'), row('trial', 'grade')]
        rows += [row('trial', 'blocked_or_unsupported', 'body_cast') for _ in range(20)]
        rows += [row('trial', 'other_' + str(i)) for i in range(10)]
        with tempfile.TemporaryDirectory() as directory:
            audit = Path(directory) / 'audit.jsonl'
            audit.write_text(''.join(json.dumps(r) + '\n' for r in rows))
            summary = trial.ground_diagnostics(audit, 'trial')
        self.assertEqual(summary['candidates'], 32)
        self.assertEqual(summary['reasons']['grade'], 2)
        self.assertEqual(summary['regions'], {'stance': 32})
        self.assertNotIn('ignored', summary['reasons'])
        self.assertEqual(summary['probeStages'], {'body_cast': 20})
        self.assertEqual(len(summary['examples']), 6)
        self.assertEqual(summary['examples'][0]['angle'], 34)
        self.assertEqual(summary['examples'][0]['roughness'], .2)
        self.assertEqual(summary['examples'][0]['region'], 'stance')
        self.assertEqual(summary['examples'][1]['collider'], 'Beech1/stem')

    def test_scouting_summary_retains_only_matching_goal_and_bounded_diagnostics(self):
        def trace(message, data, task='trial'):
            return dict(frame=dict(type='trace', source='combat', task=task, msg=message, data=data))
        rows = [trace('scout', {}, 'old'), trace('ground_selected', dict(area=[1, 2, 3], escape=[7, 2, 3], stanceRadius=3))]
        rows += [trace('scout', {})] * 20 + [trace('scout_return', {})]
        rows += [trace('scout_route', dict(returning=False)), trace('scout_route', dict(returning=True))]
        rows += [trace('scout_rejected', dict(goal=[1, 2, 3], reason='blocked_' + str(i))) for i in range(10)]
        rows += [trace('scout_rejected', dict(reason='blocked_0'))]
        with tempfile.TemporaryDirectory() as directory:
            audit = Path(directory) / 'audit.jsonl'
            audit.write_text(''.join(json.dumps(r) + '\n' for r in rows))
            summary = trial.ground_diagnostics(audit, 'trial')
        self.assertEqual(summary['candidates'], 0)
        self.assertEqual(summary['selected']['escape'], [7, 2, 3])
        self.assertEqual(summary['modes'], ['scout', 'scout_return'])
        self.assertEqual(summary['scouting']['routes'], {'outbound': 1, 'return': 1})
        self.assertEqual(summary['scouting']['reasons']['blocked_0'], 2)
        self.assertEqual(len(summary['scouting']['examples']), 6)

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
                        value = dict(status, **({'task': dict(id='trial', task='attack'), 'combat': dict(id='trial', entity='greyling', mode='kite')} if observed > 1 else {}))
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
                sent = [c for c in commands if c[0] == 'task']
                self.assertEqual(len(sent), 1)
                self.assertEqual(json.loads(sent[0][2]), {'kite': True, 'style': 'melee'})
                self.assertFalse(any(c[:2] == ['task', 'move_to'] for c in commands))
                self.assertIn(['combat', 'true'], commands)
                self.assertIn(['combat', 'false'], commands)
                self.assertEqual(commands[-3:-1], [['combat', 'false'], ['panel-start']])
                self.assertIn(['panel', 'POST', '/api/act/logout', '{}'], commands)
                summary = json.loads(next(Path(directory).glob('.checks/*/summary.json')).read_text())
                self.assertEqual(summary['outcome'], 'succeeded' if outcome == 'success' else 'aborted')
                if outcome != 'success':
                    self.assertEqual(summary['reason'], 'additional nearby creature')


if __name__ == '__main__':
    unittest.main()
