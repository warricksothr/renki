import importlib.util
import contextlib
import io
import json
from pathlib import Path
import subprocess
import tempfile
import types
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

    def test_audit_vitals_include_damage_between_polls_and_final_cleanup_state(self):
        frames = [dict(type='state', phase='menu', hp=0), dict(type='state', phase='in_world', hp=25, pos=[0, 0, 0]),
                  dict(type='event', event='damaged', amount=3),
                  dict(type='state', phase='in_world', hp=22, pos=[5, 0, 0]), dict(type='state', phase='menu', hp=0)]
        with tempfile.TemporaryDirectory() as directory:
            audit = Path(directory) / 'audit.jsonl'
            audit.write_text(''.join(json.dumps(dict(frame=f)) + '\n' for f in frames))
            self.assertEqual(trial.vital_diagnostics(audit), dict(damageEvents=1, minimumHP=22,
                             finalNativeHP=22, finalNativePos=[5, 0, 0]))

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
        rows += [trace('prepare_search', {}), trace('search_egress_route', dict(goal=[2,3,4]))]
        rows += [trace('search_egress_rejected', dict(reason='endpoint_not_dry_clear_or_separated'))]
        rows += [trace('scout', {})] * 20 + [trace('scout_return', {})]
        rows += [trace('scout_route', dict(returning=False)), trace('scout_route', dict(returning=True, trailPoints=4, returnDistance=20))]
        rows += [trace('scout_recovery', dict(reason='route changed')), trace('scout_recovery', dict(reason='route changed'), task='old')]
        rows += [trace('scout_rejected', dict(goal=[1, 2, 3], reason='blocked_' + str(i),
                                             angle=34, previousAngle=10, roughness=.2,
                                             supported=[1, 2, 3])) for i in range(10)]
        rows += [trace('scout_rejected', dict(reason='blocked_0'))]
        with tempfile.TemporaryDirectory() as directory:
            audit = Path(directory) / 'audit.jsonl'
            audit.write_text(''.join(json.dumps(r) + '\n' for r in rows))
            summary = trial.ground_diagnostics(audit, 'trial')
        self.assertEqual(summary['candidates'], 0)
        self.assertEqual(summary['selected']['escape'], [7, 2, 3])
        self.assertEqual(summary['modes'], ['prepare_search', 'scout', 'scout_return'])
        self.assertEqual(summary['scouting']['routes'], {'preparation': 1, 'outbound': 1, 'return': 1})
        self.assertEqual(summary['scouting']['reasons']['endpoint_not_dry_clear_or_separated'], 1)
        self.assertEqual(summary['scouting']['reasons']['blocked_0'], 2)
        self.assertEqual(summary['scouting']['recoveries'], {'route changed': 1})
        self.assertEqual(summary['scouting']['returnHint'], dict(trailPoints=4, returnDistance=20))
        self.assertEqual(len(summary['scouting']['examples']), 6)
        self.assertEqual(summary['scouting']['examples'][1]['angle'], 34)
        self.assertEqual(summary['scouting']['examples'][1]['supported'], [1, 2, 3])

    def test_footing_and_death_diagnostics_keep_only_latest_matching_task_evidence(self):
        def trace(message, data, task='trial'):
            return dict(frame=dict(type='trace', source='combat', task=task, msg=message, data=data))
        refusal = dict(pos=[1, 2, 3], grounded=False, swimming=False, encumbered=False, supportGap=.6)
        rows = [trace('scout_settle', {})] * 3
        rows += [trace('footing_refused', dict(supportGap=9), 'old'), trace('footing_refused', refusal)]
        rows += [trace('death_observed', dict(entity='selected', evidence='owner_callback', ignored='omit')),
                 trace('death_observed', dict(entity='other'), 'old')]
        with tempfile.TemporaryDirectory() as directory:
            audit = Path(directory) / 'audit.jsonl'
            audit.write_text(''.join(json.dumps(r) + '\n' for r in rows))
            summary = trial.ground_diagnostics(audit, 'trial')
        self.assertEqual(summary['modes'], ['scout_settle'])
        self.assertEqual(summary['footing'], dict(settles=3, lastRefusal=refusal))
        self.assertEqual(summary['death'], dict(entity='selected', evidence='owner_callback'))

    def test_direct_mode_requires_an_explicit_target_and_no_lure_or_harvest_options(self):
        for arguments in (['--direct'], ['--direct', '--entity', 'greyling', '--at', '0', '0', '0'], ['--direct', '--entity', 'greyling', '--harvest-tree']):
            with self.subTest(arguments=arguments), mock.patch.object(trial.sys, 'argv', ['combat-trial', *arguments]), mock.patch.object(trial.subprocess, 'run') as run, contextlib.redirect_stderr(io.StringIO()):
                with self.assertRaises(SystemExit):
                    trial.main()
                run.assert_not_called()

    def test_runner_stops_and_restores_gate_after_success_or_additional_attacker(self):
        for outcome in ('success', 'extra attacker', 'direct'):
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
                        value = [dict(frame=dict(re='trial', status='succeeded'))] if outcome in ('success', 'direct') else []
                    elif args[0] == 'audit-window':
                        value = '/saved/audit.jsonl'
                    return subprocess.CompletedProcess(command, 0, json.dumps(value) if isinstance(value, (dict, list)) else value, '')

                with mock.patch.object(trial, 'ROOT', Path(directory)), mock.patch.object(trial.subprocess, 'run', side_effect=run), mock.patch.object(trial.sys, 'argv', ['combat-trial', '--wait-s', '0', *(['--direct', '--entity', 'greyling'] if outcome == 'direct' else [])]), contextlib.redirect_stdout(io.StringIO()):
                    code = trial.main()
                self.assertEqual(code, 0 if outcome in ('success', 'direct') else 1)
                self.assertEqual(commands.count(['stop']), 2)
                sent = [c for c in commands if c[0] == 'task']
                self.assertEqual(len(sent), 1)
                self.assertEqual(json.loads(sent[0][2]), {'kite': False, 'style': 'melee', 'entity': 'greyling'} if outcome == 'direct' else {'kite': True, 'style': 'melee'})
                self.assertFalse(any(c[:2] == ['task', 'move_to'] for c in commands))
                self.assertIn(['combat', 'true'], commands)
                self.assertIn(['combat', 'false'], commands)
                self.assertEqual(commands[-3:-1], [['combat', 'false'], ['panel-start']])
                self.assertIn(['panel', 'POST', '/api/act/logout', '{}'], commands)
                summary = json.loads(next(Path(directory).glob('.checks/*/summary.json')).read_text())
                self.assertEqual(summary['outcome'], 'succeeded' if outcome in ('success', 'direct') else 'aborted')
                if outcome == 'extra attacker':
                    self.assertEqual(summary['reason'], 'additional nearby creature')

    def test_resupply_waits_for_startup_home_but_stops_on_damage_or_takeover(self):
        import types
        for mode in ('finished', 'damage', 'owner', 'connection'):
            with self.subTest(mode=mode), tempfile.TemporaryDirectory() as directory:
                observed, commands = 0, []
                audit = Path(directory) / 'audit.jsonl'
                audit.write_text('')
                status = dict(self.status, world='one', connection=1, home=dict(world='one', pos=[0, 32, 0]),
                              food=dict(active=[]))

                def run(command, **kwargs):
                    nonlocal observed
                    args = command[2:]
                    commands.append(args)
                    value = ''
                    if args[:3] == ['panel', 'GET', '/api/status']:
                        observed += 1
                        value = dict(status)
                        if observed == 1:
                            value['task'] = dict(id='home', task='move_to')
                        elif mode == 'damage':
                            value['hp'] = 77
                        elif mode == 'owner':
                            value['task'] = dict(id='owner', task='follow')
                        elif mode == 'connection':
                            value['connection'] = 2
                    elif args[:3] == ['panel', 'GET', '/api/map']:
                        value = dict(perception={})
                    elif args[:2] == ['panel', 'GET'] and args[2].startswith('/api/task?'):
                        value = []
                    elif args[0] == 'audit-window':
                        value = str(audit)
                    return subprocess.CompletedProcess(command, 0, json.dumps(value) if isinstance(value, (dict, list)) else value, '')

                prepare = mock.Mock(side_effect=lambda summary, *a: summary.update(outcome='succeeded'))
                module = types.SimpleNamespace(startup_home=lambda *a: True, run=prepare)
                with mock.patch.object(trial, 'ROOT', Path(directory)), mock.patch.object(trial.subprocess, 'run', side_effect=run), mock.patch.object(trial.sys, 'argv', ['combat-trial', '--resupply-food']), mock.patch.dict(trial.sys.modules, resupply_trial=module), mock.patch.object(trial.time, 'sleep'), contextlib.redirect_stdout(io.StringIO()):
                    code = trial.main()
                self.assertEqual(code, 0 if mode == 'finished' else 1)
                self.assertEqual(prepare.call_count, 1 if mode == 'finished' else 0)
                self.assertFalse(any(c[0] == 'task' for c in commands))
                self.assertIn(['panel', 'POST', '/api/act/logout', '{}'], commands)

    def test_gather_dispatch_proof_and_cleanup_survive_partial_failures(self):
        for mode in ('success', 'lost_save_response', 'bad_proof', 'missing_audit', 'missing_proof', 'cleanup_failure', 'audit_damage'):
            with self.subTest(mode=mode), tempfile.TemporaryDirectory() as directory:
                commands = []
                audit = Path(directory) / 'audit.jsonl'
                audit.write_text(json.dumps(dict(frame=dict(type='event', event='damaged', amount=1))) + '\n' if mode == 'audit_damage' else '')
                status = dict(self.status, food=dict(active=[dict(prefab='CookedDeerMeat')]))

                def run(command, **kwargs):
                    args = command[2:]
                    commands.append(args)
                    value = ''
                    if args[:3] == ['panel', 'GET', '/api/status']:
                        value = status
                    elif args[:3] == ['panel', 'GET', '/api/map']:
                        value = dict(perception={})
                    elif args[0] == 'audit-window':
                        value = str(audit) if mode != 'missing_audit' else str(audit.parent / 'missing')
                    elif mode == 'cleanup_failure' and args[:3] == ['panel', 'POST', '/api/behaviours'] and json.loads(args[3])['action'] == 'stop':
                        return subprocess.CompletedProcess(command, 1, '', 'stop response lost')
                    return subprocess.CompletedProcess(command, 0, json.dumps(value) if isinstance(value, (dict, list)) else value, '')

                def gather_run(summary, call, observe, saved):
                    summary['behaviourName'] = 'unique-test'
                    if mode == 'lost_save_response':
                        raise RuntimeError('save response lost')
                    summary['outcome'] = 'awaiting_proof'

                def verify(path, summary):
                    if mode == 'bad_proof':
                        raise RuntimeError('missing receipt')
                    return {} if mode in ('missing_proof', 'lost_save_response') else dict(outcome='succeeded')

                module = types.SimpleNamespace(run=mock.Mock(side_effect=gather_run), verify=mock.Mock(side_effect=verify))
                with mock.patch.object(trial, 'ROOT', Path(directory)), mock.patch.object(trial.subprocess, 'run', side_effect=run), mock.patch.object(trial.sys, 'argv', ['combat-trial', '--gather-tree', '--wait-s', '0']), mock.patch.dict(trial.sys.modules, gather_trial=module), contextlib.redirect_stdout(io.StringIO()):
                    code = trial.main()
                self.assertEqual(code, 0 if mode == 'success' else 1)
                module.run.assert_called_once()
                self.assertNotIn(['combat', 'true'], commands)
                self.assertFalse(any(c[0] == 'task' for c in commands))
                actions = [json.loads(c[3]) for c in commands if c[:3] == ['panel', 'POST', '/api/behaviours']]
                self.assertEqual(actions, [dict(action=a, name='unique-test') for a in ('stop', 'delete')])
                self.assertLess(next(i for i, c in enumerate(commands) if c[:3] == ['panel', 'POST', '/api/behaviours']), commands.index(['camera']))
                self.assertEqual(commands[-5:-1], [['panel', 'POST', '/api/act/logout', '{}'], ['stop'], ['combat', 'false'], ['panel-start']])
                summary = json.loads(next(Path(directory).glob('.checks/*/summary.json')).read_text())
                self.assertNotEqual(summary['outcome'], 'awaiting_proof')
                if mode == 'missing_audit':
                    module.verify.assert_not_called()


if __name__ == '__main__':
    unittest.main()
