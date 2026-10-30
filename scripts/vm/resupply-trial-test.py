"""Food resupply needs exact transfers, current scope and observed consumption."""
import copy
import importlib.util
from pathlib import Path
import tempfile
import unittest

spec = importlib.util.spec_from_file_location('resupply', Path(__file__).with_name('resupply_trial.py'))
resupply = importlib.util.module_from_spec(spec)
spec.loader.exec_module(resupply)


class Resupply(unittest.TestCase):
    def status(self):
        return dict(connected=True, phase='in_world', world='one', connection=1, hp=25, hpMax=25,
                    pos=[0, 0, 0], home=dict(world='one', pos=[0, 0, 0]), food=dict(active=[]))

    def mapped(self):
        return dict(containers=[dict(entity='chest', pos=[1, 0, 0], items=[dict(prefab='Honey', stack=8), dict(prefab='Raspberry', stack=8), dict(prefab='Wood', stack=50)])])

    def test_short_home_approach_runs_only_with_stamina(self):
        status = dict(self.status(), pos=[30, 0, 0], stamina=40)
        self.assertTrue(resupply.home_run(status))
        self.assertFalse(resupply.home_run(dict(status, stamina=10)))
        self.assertFalse(resupply.home_run(dict(status, pos=[375, 0, 0])))

    def test_selection_is_home_scoped_bounded_and_excludes_arbitrary_items(self):
        mapped = self.mapped()
        mapped['containers'].append(dict(entity='far', pos=[100, 0, 0], items=[dict(prefab='CookedDeerMeat', stack=10)]))
        found = resupply.candidates(self.status(), mapped, {'Honey': 4})
        self.assertEqual([(x[2], x[3], x[4]) for x in found], [('chest', 'Raspberry', 4)])
        with self.assertRaisesRegex(RuntimeError, 'no known home'):
            resupply.candidates(dict(self.status(), home={}), mapped, {})

    def test_resupply_requires_task_matched_receipts_and_fresh_inventory(self):
        for mode in ('valid', 'wrong_receipt', 'stale_inventory', 'damaged', 'new_connection'):
            with self.subTest(mode=mode), tempfile.TemporaryDirectory() as folder:
                status, mapped, requests = self.status(), self.mapped(), []
                inventory = dict(seq=1, size=[8, 4], items=[])
                active_task, history = None, []

                def observe():
                    st = copy.deepcopy(status)
                    if requests and mode == 'damaged':
                        st['hp'] = 24
                    if requests and mode == 'new_connection':
                        st['connection'] = 2
                    return st, mapped

                def call(*args, **kwargs):
                    nonlocal active_task, history
                    if args[:3] == ('panel', 'GET', '/api/inventory'):
                        return dict(world='one', connection=2 if mode == 'stale_inventory' else 1, inventory=copy.deepcopy(inventory))
                    if args[0] == 'task':
                        import json
                        self.assertEqual(args[1], 'withdraw')
                        request = json.loads(args[2]); requests.append(request)
                        active_task = 't-' + str(len(requests))
                        prefab = request['items'][0]['prefab']
                        inventory['seq'] += 1
                        inventory['items'].append(dict(prefab=prefab, stack=3))
                        status['food']['active'].append(dict(prefab=prefab))
                        history = [dict(frame=dict(type='container', entity='chest', task='wrong' if mode == 'wrong_receipt' else active_task,
                                                   direction='withdraw', moved={prefab: 4})),
                                   dict(frame=dict(type='task', re=active_task, status='succeeded'))]
                        return dict(id=active_task)
                    if args[:2] == ('panel', 'GET') and args[2].startswith('/api/task?'):
                        return history
                    raise AssertionError(args)

                def terminal(rows, task_id):
                    return next(r['frame'] for r in rows if r['frame'].get('type') == 'task' and r['frame']['re'] == task_id)

                summary = {}
                if mode == 'valid':
                    resupply.run(summary, call, observe, terminal, Path(folder))
                    self.assertEqual(summary['outcome'], 'succeeded')
                    self.assertEqual(summary['supplied'], dict(Honey=4, Raspberry=4))
                    self.assertEqual(len(requests), 2)
                else:
                    with self.assertRaises(RuntimeError):
                        resupply.run(summary, call, observe, terminal, Path(folder))
                    self.assertNotEqual(summary.get('outcome'), 'succeeded')


if __name__ == '__main__':
    unittest.main()
