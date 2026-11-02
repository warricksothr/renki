"""Food resupply needs exact transfers, current scope and observed consumption."""
import copy
import importlib.util
import itertools
from pathlib import Path
import tempfile
import unittest
from unittest import mock

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

    def test_startup_wait_requires_exact_outgoing_home_destination_and_task(self):
        status = dict(self.status(), task=dict(id='home', task='move_to'))
        row = dict(dir='out', frame=dict(type='task', id='home', task='move_to', args=dict(to=[0, 0, 0])))
        self.assertTrue(resupply.startup_home(status, [row]))
        self.assertTrue(resupply.startup_home(status, dict(value=[row])))
        for bad in [dict(row, dir='in'), dict(row, frame=dict(row['frame'], id='other')),
                    dict(row, frame=dict(row['frame'], args=dict(to=[1, 0, 0])))]:
            self.assertFalse(resupply.startup_home(status, [bad]))
        self.assertFalse(resupply.startup_home(dict(status, task=dict(id='home', task='follow')), [row]))

    def test_selection_is_home_scoped_bounded_and_excludes_arbitrary_items(self):
        mapped = self.mapped()
        mapped['containers'].append(dict(entity='far', pos=[100, 0, 0], items=[dict(prefab='CookedDeerMeat', stack=10)]))
        found = resupply.candidates(self.status(), mapped, {'Honey': 4})
        self.assertEqual([(x[2], x[3], x[4]) for x in found], [('chest', 'Raspberry', 4)])
        with self.assertRaisesRegex(RuntimeError, 'no known home'):
            resupply.candidates(dict(self.status(), home={}), mapped, {})

    def test_stale_chest_hint_can_select_only_a_unique_live_position_match(self):
        mapped = self.mapped()
        mapped['perception'] = dict(structures=[dict(id='new', kind='container', pos=[1, 0, 0])])
        self.assertEqual({c[2] for c in resupply.candidates(self.status(), mapped, {})}, {'new'})
        mapped['perception']['structures'].append(dict(id='ambiguous', kind='container', pos=[1.1, 0, 0]))
        self.assertTrue(all(c[0] == len(resupply.FOODS) for c in resupply.candidates(self.status(), mapped, {})))
        mapped['perception']['structures'] = [dict(id='upstairs', kind='container', pos=[1, 3, 0])]
        self.assertEqual({c[2] for c in resupply.candidates(self.status(), mapped, {}) if c[0] < len(resupply.FOODS)}, {'chest'})
        mapped['perception']['structures'].append(dict(id='chest', kind='container', pos=[1, 0, 0]))
        self.assertEqual({c[2] for c in resupply.candidates(self.status(), mapped, {}) if c[0] < len(resupply.FOODS)}, {'chest'})

    def test_opened_contents_override_hoped_for_food_and_other_boxes_are_candidates(self):
        mapped = self.mapped()
        mapped['perception'] = dict(structures=[dict(id='chest', kind='container', pos=[1, 0, 0]),
                                                dict(id='another', kind='container', pos=[2, 0, 0])])
        opened = dict(chest=dict(entity='chest', items=[dict(prefab='Stone', stack=20)]))
        choices = resupply.candidates(self.status(), mapped, {}, opened)
        self.assertEqual([(c[2], c[3], c[4]) for c in choices], [('another', 'CookedDeerMeat', 1)])
        opened['chest']['items'].append(dict(prefab='Blueberries', stack=8))
        choices = resupply.candidates(self.status(), mapped, {}, opened)
        self.assertEqual(choices[0][2:], ('chest', 'Blueberries', 4))

    def test_resupply_requires_task_matched_receipts_and_fresh_inventory(self):
        for mode in ('valid', 'carried_food', 'return_completed_after_poll', 'new_order_after_return', 'stale_completion', 'wrong_receipt', 'stale_inventory', 'damaged', 'new_connection'):
            with self.subTest(mode=mode), tempfile.TemporaryDirectory() as folder:
                status, mapped, requests = self.status(), self.mapped(), []
                if mode in ('return_completed_after_poll', 'new_order_after_return', 'stale_completion'):
                    status['pos'] = [8, 0, 0]
                inventory = dict(seq=1, size=[8, 4], items=[])
                if mode == 'carried_food':
                    inventory['items'] = [dict(prefab='CookedDeerMeat', stack=4)]
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
                        if args[1] == 'move_to':
                            self.assertTrue(json.loads(args[2])['dry'])
                            active_task = 'home'
                            status['task'] = dict(id='home', task='move_to')
                            history = [dict(frame=dict(type='task', re='home', status='succeeded'))]
                            return dict(id='home')
                        self.assertEqual(args[1], 'withdraw')
                        request = json.loads(args[2]); requests.append(request)
                        active_task = 't-' + str(len(requests))
                        prefab = request['items'][0]['prefab']
                        inventory['seq'] += 1
                        inventory['items'].append(dict(prefab=prefab, stack=3))
                        status['food']['active'].append(dict(prefab=prefab))
                        history = [dict(frame=dict(type='container', entity='chest', pos=[1, 0, 0],
                                                   items=[dict(prefab='Honey', stack=4), dict(prefab='Raspberry', stack=8)],
                                                   task='wrong' if mode == 'wrong_receipt' else active_task,
                                                   direction='withdraw', moved={prefab: 4})),
                                   dict(frame=dict(type='task', re=active_task, status='succeeded'))]
                        return dict(id=active_task)
                    if args[:2] == ('panel', 'GET') and args[2].startswith('/api/task?'):
                        if active_task == 'home' and mode != 'stale_completion':
                            # The preceding status still reported travelling.
                            status['pos'] = [0, 0, 0]
                            status['task'] = dict(id='new-order', task='follow') if mode == 'new_order_after_return' else None
                        return history
                    raise AssertionError(args)

                def terminal(rows, task_id):
                    return next(r['frame'] for r in rows if r['frame'].get('type') == 'task' and r['frame']['re'] == task_id)

                summary = {}
                if mode in ('valid', 'carried_food', 'return_completed_after_poll'):
                    resupply.run(summary, call, observe, terminal, Path(folder))
                    self.assertEqual(summary['outcome'], 'succeeded')
                    self.assertEqual(summary['supplied'], dict(Honey=4) if mode == 'carried_food' else dict(Honey=4, Raspberry=4))
                    self.assertEqual(len(requests), 1 if mode == 'carried_food' else 2)
                elif mode == 'stale_completion':
                    with mock.patch.object(resupply.time, 'monotonic', side_effect=itertools.count()), mock.patch.object(resupply.time, 'sleep'):
                        with self.assertRaisesRegex(RuntimeError, 'completion state remained stale'):
                            resupply.run(summary, call, observe, terminal, Path(folder))
                    self.assertFalse(requests)
                else:
                    with self.assertRaises(RuntimeError):
                        resupply.run(summary, call, observe, terminal, Path(folder))
                    self.assertNotEqual(summary.get('outcome'), 'succeeded')

    def test_search_covers_each_unique_area_chest_with_no_fixed_count_cutoff(self):
        for boxes, has_food in ((7, True), (14, True), (14, False)):
            with self.subTest(boxes=boxes, has_food=has_food), tempfile.TemporaryDirectory() as folder:
                status, inventory = self.status(), dict(seq=1, size=[8, 4], items=[])
                structures = [dict(id=str(i), kind='container', pos=[.5+i*.4, 0, 0]) for i in range(boxes)]
                mapped = dict(perception=dict(structures=structures), containers=[])
                requests, history = [], []

                def call(*args, **kwargs):
                    nonlocal history
                    if args[:3] == ('panel', 'GET', '/api/inventory'):
                        return dict(world='one', connection=1, inventory=copy.deepcopy(inventory))
                    if args[0] == 'task':
                        import json
                        request = json.loads(args[2]); requests.append(request)
                        chest, prefab = request['container'], request['items'][0]['prefab']
                        stocked = has_food and chest == str(boxes-1)
                        found = stocked and prefab == 'Honey'
                        if found:
                            inventory.update(seq=2, items=[dict(prefab='Honey', stack=3)])
                            status['food']['active'] = [dict(prefab='Honey')]
                        task_id = str(len(requests))
                        receipt = dict(type='container', entity=chest, task=task_id, direction='withdraw', pos=structures[int(chest)]['pos'],
                                       items=[dict(prefab='Honey' if stocked else 'Stone', stack=4)],
                                       moved=dict(Honey=4) if found else {})
                        history = [dict(frame=receipt), dict(frame=dict(type='task', re=task_id,
                                   status='succeeded' if found else 'failed', reason='invalid_target'))]
                        return dict(id=task_id)
                    if args[:2] == ('panel', 'GET') and args[2].startswith('/api/task?'):
                        return history
                    raise AssertionError(args)

                summary = {}
                def run():
                    resupply.run(summary, call, lambda: (copy.deepcopy(status), mapped),
                                 lambda rows, task: rows[-1]['frame'], Path(folder))
                if has_food:
                    run()
                    self.assertEqual(summary['outcome'], 'succeeded')
                    self.assertEqual(summary['checkedChests'], boxes)
                    self.assertEqual(len(requests), boxes+1) # Inspect, then withdraw the observed Honey.
                else:
                    with self.assertRaisesRegex(RuntimeError, 'no verified food transfer'):
                        run()
                    self.assertEqual(len(requests), boxes)
                    self.assertEqual(summary['checkedChests'], boxes)
                self.assertEqual(summary['searchChests'], boxes)
                pairs = [(r['container'], r['items'][0]['prefab']) for r in requests]
                self.assertEqual(len(pairs), len(set(pairs)))

    def test_missing_food_refreshes_the_hint_and_searches_another_box(self):
        status = self.status()
        mapped = dict(containers=[dict(entity='wrong', pos=[1, 0, 0], items=[dict(prefab='Honey', stack=4)])],
                      perception=dict(structures=[dict(id='wrong', kind='container', pos=[1, 0, 0]),
                                                  dict(id='food', kind='container', pos=[2, 0, 0])]))
        inventory, requests, history = dict(seq=1, size=[8, 4], items=[]), [], []

        def call(*args, **kwargs):
            nonlocal history
            if args[:3] == ('panel', 'GET', '/api/inventory'):
                return dict(world='one', connection=1, inventory=copy.deepcopy(inventory))
            if args[0] == 'task':
                import json
                request = json.loads(args[2])
                chest, prefab = request['container'], request['items'][0]['prefab']
                requests.append((chest, prefab))
                task = 't-' + str(len(requests))
                found = chest == 'food' and prefab == 'Honey'
                if found:
                    inventory.update(seq=2, items=[dict(prefab='Honey', stack=3)])
                    status['food']['active'] = [dict(prefab='Honey')]
                history = [dict(frame=dict(type='container', entity=chest, task=task, direction='withdraw',
                                           pos=[1 if chest == 'wrong' else 2, 0, 0],
                                           items=[dict(prefab='Stone' if chest == 'wrong' else 'Honey', stack=20 if chest == 'wrong' else 4)],
                                           moved=dict(Honey=4) if found else {})),
                           dict(frame=dict(type='task', re=task, status='succeeded' if found else 'failed', reason='invalid_target'))]
                return dict(id=task)
            if args[:2] == ('panel', 'GET') and args[2].startswith('/api/task?'):
                return history
            raise AssertionError(args)

        with tempfile.TemporaryDirectory() as folder:
            summary = {}
            resupply.run(summary, call, lambda: (copy.deepcopy(status), mapped),
                         lambda rows, task: rows[-1]['frame'], Path(folder))
        self.assertEqual(requests, [('wrong', 'Honey'), ('food', 'CookedDeerMeat'), ('food', 'Honey')])
        self.assertEqual(summary['outcome'], 'succeeded')
        self.assertEqual(summary['supplied'], dict(Honey=4))


if __name__ == '__main__':
    unittest.main()
