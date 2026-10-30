import contextlib
import importlib.util
import io
import json
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest import mock

import harvest_trial as harvest

spec = importlib.util.spec_from_file_location('trial', Path(__file__).with_name('combat-trial.py'))
trial = importlib.util.module_from_spec(spec)
spec.loader.exec_module(trial)


class HarvestTrialTests(unittest.TestCase):
    def setUp(self):
        self.status = dict(connected=True, phase='in_world', world='world', connection=1,
                           hp=80, hpMax=80, stamina=60, pos=[0, 32, 0], food=dict(active=[dict(prefab='CookedDeerMeat')]))
        self.tree = dict(id='tree', kind='tree', prefab='Beech1', minTool=0, pos=[5, 32, 0])
        self.log = dict(id='log', kind='log', prefab='beech_log', minTool=0, pos=[5, 32, 1])
        self.axe = dict(prefab='AxeFlint', stack=1, durability=92, durabilityMax=100)
        self.pack = dict(world='world', connection=1, weight=8, weightMax=300,
                         inventory=dict(size=[8, 4], items=[self.axe, dict(prefab='Wood', stack=2)]))

    def test_selection_requires_full_low_tier_tree_and_clear_characters_and_structures(self):
        perception = dict(resources=[self.tree, dict(self.tree, id='small', prefab='Beech_small1'),
                                     dict(self.tree, id='birch', prefab='Birch1', minTool=1)])
        self.assertEqual(harvest.select_resource(self.status, perception)['id'], 'tree')
        for occupied in (dict(characters=[dict(pos=[6, 32, 0], kind='player', pvp=False)]),
                         dict(characters=[dict(pos=[25, 32, 0], kind='creature')]),
                         dict(characters=[dict(pos=[6, 32, 0], kind='creature', hostile=False)]),
                         dict(structures=[dict(pos=[7, 32, 0])])):
            self.assertIsNone(harvest.select_resource(self.status, dict(perception, **occupied)))
        # A nearby standing tree needs the full falling exclusion; splitting
        # a settled log does not inherit that tall-tree radius.
        self.assertTrue(harvest.worksite_clear(self.status, dict(characters=[dict(pos=[25, 32, 0])]), self.log['pos'], False))

    def test_only_new_nearby_same_family_logs_are_eligible_and_unsafe_logs_remain_unfinished(self):
        old = dict(self.log, id='old')
        far = dict(self.log, id='far', pos=[50, 32, 0])
        other = dict(self.log, id='pine', prefab='Pine_log')
        perception = dict(resources=[old, far, other, self.log])
        self.assertEqual(harvest.select_resource(self.status, perception, {'old'}, self.tree['pos'])['id'], 'log')
        perception['characters'] = [dict(pos=[5, 32, 1], kind='player')]
        self.assertIsNone(harvest.select_resource(self.status, perception, {'old'}, self.tree['pos']))
        self.assertEqual([r['id'] for r in harvest.remaining_logs(perception, {'old'}, self.tree['pos'], set())], ['log'])

    def test_pack_needs_real_axe_durability_free_slots_and_spare_weight(self):
        self.assertTrue(harvest.pack_ready(self.pack))
        self.assertFalse(harvest.pack_ready(dict(self.pack, weight=250)))
        self.assertFalse(harvest.pack_ready(dict(self.pack, inventory=dict(size=[1, 1], items=[self.axe]))))
        self.assertFalse(harvest.pack_ready(dict(self.pack, inventory=dict(size=[8, 4], items=[dict(self.axe, durability=0)]))))
        self.assertFalse(harvest.pack_ready(dict(self.pack, inventory=None)))

    def test_diagnostics_scope_and_bound_repeated_contacts_and_report_saturation(self):
        def trace(task, message, **data):
            return dict(frame=dict(type='trace', source='harvest', task=task, msg=message, data=data))
        rows = [trace('old', 'resource broke after local hit', hits=99)]
        rows += [trace('trial', 'approaching physical contact', goal=[i, 0, 0]) for i in range(200)]
        rows += [trace('trial', 'safety scan', phase='protection', capacity=2048, count=2048, complete=False)]
        rows += [dict(frame=dict(type='event', event='damaged'))]
        rows += [dict(frame=dict(type='state', task=dict(id='trial', task='harvest')))]
        with tempfile.TemporaryDirectory() as directory:
            audit = Path(directory) / 'audit.jsonl'
            audit.write_text(''.join(json.dumps(row) + '\n' for row in rows))
            value = harvest.diagnostics(audit, ['trial'])
        self.assertEqual(list(value['tasks']), ['trial'])
        self.assertEqual(value['damageEvents'], 1)
        task = value['tasks']['trial']
        self.assertEqual(task['messages']['approaching physical contact'], 200)
        self.assertEqual(task['last']['approaching physical contact']['goal'], [199, 0, 0])
        self.assertFalse(task['last']['safety scan']['complete'])
        self.assertNotIn('resource broke after local hit', task['last'])
        self.assertLess(len(json.dumps(value)), 600)

    def test_native_goals_evidence_and_cleanup_for_success_and_failures(self):
        for scenario in ('success', 'logs_only', 'no_site', 'failed', 'missing_hit', 'world_change', 'damage', 'pack_scope', 'no_wood', 'unsafe_log'):
            with self.subTest(scenario=scenario), tempfile.TemporaryDirectory() as directory:
                commands, sent = [], []
                resources = [self.tree, dict(self.log, id='old')]
                if scenario == 'no_site':
                    resources = []
                if scenario == 'logs_only':
                    resources = [self.log]
                active, wood, clock = None, 2, [0.0]
                pack = json.loads(json.dumps(self.pack))

                def run(command, **kwargs):
                    nonlocal active, wood, resources
                    args = command[2:]
                    commands.append(args)
                    value = ''
                    if args[:3] == ['panel', 'GET', '/api/status']:
                        value = dict(self.status)
                        if active:
                            value['task'] = dict(id=active, task='harvest')
                            if scenario == 'world_change':
                                value['world'] = 'elsewhere'
                            if scenario == 'damage':
                                value['hp'] = 79
                    elif args[:3] == ['panel', 'GET', '/api/map']:
                        perception = dict(resources=resources, characters=[], structures=[])
                        if scenario == 'unsafe_log' and sent:
                            perception['characters'] = [dict(pos=[5, 32, 1], kind='player')]
                        value = dict(world='world', connection=1, perception=perception)
                    elif args[:3] == ['panel', 'GET', '/api/inventory']:
                        pack['inventory']['items'][1]['stack'] = wood
                        value = pack
                        if scenario == 'pack_scope':
                            value = dict(pack, connection=2)
                    elif args[0] == 'task':
                        self.assertEqual(args[1], 'harvest')
                        goal = json.loads(args[2]); sent.append(goal)
                        active = 'h-' + str(len(sent))
                        value = dict(id=active)
                    elif args[:2] == ['panel', 'GET'] and args[2].startswith('/api/task?'):
                        task_id = active
                        failed = scenario == 'failed'
                        value = [dict(frame=dict(re=task_id, status='failed' if failed else 'succeeded', reason='refused' if failed else None))]
                        if scenario != 'missing_hit':
                            value.append(dict(frame=dict(type='trace', task=task_id, source='harvest',
                                                         msg='resource broke after local hit', data=dict(hits=3))))
                        resource_id = sent[-1]['entity']
                        resources = [r for r in resources if r['id'] != resource_id]
                        if resource_id == 'tree':
                            resources.append(self.log)
                        elif resource_id == 'log':
                            resources.extend([dict(self.log, id='half1'), dict(self.log, id='half2')])
                        elif scenario != 'no_wood':
                            wood += 10
                        active = None
                    elif args[0] == 'audit-window':
                        audit = Path(directory) / 'audit.jsonl'; audit.write_text('')
                        value = str(audit)
                    return subprocess.CompletedProcess(command, 0, json.dumps(value) if isinstance(value, (dict, list)) else value, '')

                def sleep(seconds):
                    clock[0] += seconds

                mode = ['--harvest-logs', '5', '32', '0'] if scenario == 'logs_only' else ['--harvest-tree']
                with mock.patch.object(trial, 'ROOT', Path(directory)), mock.patch.object(trial.subprocess, 'run', side_effect=run), \
                        mock.patch.object(trial.sys, 'argv', ['combat-trial', *mode, '--wait-s', '0']), \
                        mock.patch.object(harvest.time, 'monotonic', side_effect=lambda: clock[0]), \
                        mock.patch.object(harvest.time, 'sleep', side_effect=sleep), contextlib.redirect_stdout(io.StringIO()):
                    code = trial.main()
                self.assertEqual(code, 0 if scenario in ('success', 'logs_only') else 1)
                self.assertEqual(commands.count(['stop']), 2)
                self.assertEqual(commands[-3], ['combat', 'false'])
                self.assertEqual(commands[-2], ['panel-start'])
                self.assertNotIn(['combat', 'true'], commands)
                self.assertIn(['panel', 'POST', '/api/act/logout', '{}'], commands)
                self.assertTrue(all(g['collect'] and g['entity'] != 'old' for g in sent))
                if scenario == 'success':
                    self.assertEqual([g['entity'] for g in sent], ['tree', 'log', 'half1', 'half2'])
                    saved = next((Path(directory) / '.checks').glob('harvest-trial-*/summary.json'))
                    result = json.loads(saved.read_text())
                    self.assertEqual(result['woodGain'], 20)
                    self.assertEqual(result['logsHarvested'], 3)
                if scenario == 'logs_only':
                    self.assertEqual([g['entity'] for g in sent], ['log', 'half1', 'half2'])
                    saved = next((Path(directory) / '.checks').glob('harvest-trial-*/summary.json'))
                    result = json.loads(saved.read_text())
                    self.assertEqual(result['mode'], 'logs_only')
                    self.assertNotIn('tree', result)
                    self.assertIn('full-tree benchmark not tested', result['result']['detail'])


if __name__ == '__main__':
    unittest.main()
