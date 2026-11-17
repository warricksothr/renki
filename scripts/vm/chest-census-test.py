#!/usr/bin/env python3
"""Offline checks of chest-census.py's selection, ordering and summary."""
import importlib.util
from pathlib import Path
import unittest

spec = importlib.util.spec_from_file_location('census', Path(__file__).with_name('chest-census.py'))
census = importlib.util.module_from_spec(spec)
spec.loader.exec_module(census)

CHEST = 'piece_chest_wood'


class CensusTest(unittest.TestCase):
    def test_one_chest_under_three_ids_is_one_candidate_asked_by_its_seen_id(self):
        # Seen live on 2026-11-17: remembered 1:15433, perceived as 1:15527.
        remembered = [{'entity': '1:15433', 'prefab': CHEST, 'pos': [-170.71, 30.97, -465.33], 'ageS': 99,
                       'items': [{'prefab': 'Wood'}]}]
        seen = [{'id': '1:15527', 'prefab': CHEST, 'pos': [-170.71, 30.97, -465.33]}]
        marks = [{'entity': '1:15527', 'prefab': CHEST, 'pos': [-170.5, 31, -465.5]}]
        found = census.candidates(remembered, marks, seen, [-170, 31, -468], 40)
        self.assertEqual(len(found), 1)
        self.assertEqual((found[0]['entity'], found[0]['opened'], found[0]['items']), ('1:15527', True, 1))
        self.assertEqual(found[0]['ids'], ['1:15433', '1:15527'])

    def test_observed_entity_is_superseded_and_only_a_fallback(self):
        # internal/home/containers.go: ObservedEntity is the old object the
        # contents came from; Entity is the current one at that place.
        remembered = [{'entity': '1:15687', 'observedEntity': '1:15597', 'prefab': 'piece_chest_barrel',
                       'pos': [0, 0, 0], 'items': []}]
        found = census.candidates(remembered, [], [], [0, 0, 0], 10)
        self.assertEqual((found[0]['entity'], found[0]['ids']), ('1:15687', ['1:15687', '1:15597']))

    def test_stacked_and_adjacent_chests_stay_separate(self):
        seen = [{'id': 'a', 'prefab': CHEST, 'pos': [0, 31, 0]}, {'id': 'b', 'prefab': CHEST, 'pos': [0, 32, 0]},
                {'id': 'c', 'prefab': CHEST, 'pos': [0.8, 31, 0]}]
        self.assertEqual(len(census.candidates([], [], seen, [0, 31, 0], 10)), 3)

    def test_other_structures_far_chests_and_bad_rows_are_left_out(self):
        seen = [{'id': 'fire', 'prefab': 'Candle_resin', 'pos': [0, 0, 0]},
                {'id': 'far', 'prefab': CHEST, 'pos': [400, 0, 0]},
                {'id': 'nopos', 'prefab': CHEST}, {'prefab': CHEST, 'pos': [0, 0, 0]}]
        self.assertEqual(census.candidates([], [], seen, [0, 0, 0], 40), [])

    def test_any_container_kind_counts_whatever_its_prefab(self):
        seen = [{'id': 'barrel', 'kind': 'container', 'prefab': 'piece_chest_barrel', 'pos': [0, 0, 0]},
                {'id': 'iron', 'kind': 'container', 'prefab': 'piece_chest', 'pos': [2, 0, 0]},
                {'id': 'cart', 'kind': 'container', 'prefab': 'Cart', 'pos': [4, 0, 0]},
                {'id': 'sign', 'kind': 'structure', 'prefab': 'piece_chest_sign', 'pos': [6, 0, 0]},
                {'id': 'bare', 'prefab': 'piece_chest_blackmetal', 'pos': [8, 0, 0]}]
        found = census.candidates([], [], seen, [0, 0, 0], 40)
        self.assertEqual([c['entity'] for c in found], ['barrel', 'iron', 'cart', 'bare'])

    def test_order_walks_to_the_nearest_next_chest(self):
        chests = [{'entity': name, 'pos': [x, 0, 0]} for name, x in (('far', 9), ('near', 1), ('mid', 4))]
        self.assertEqual([c['entity'] for c in census.order(chests, [0, 0, 0])], ['near', 'mid', 'far'])

    def test_unsighted_landmarks_come_last_and_gone_ones_are_not_refusals(self):
        marks = [{'entity': 'old', 'kind': 'container', 'pos': [1, 0, 0], 'ageS': 14 * 86400},
                 {'entity': 'today', 'kind': 'container', 'pos': [5, 0, 0], 'ageS': 600},
                 {'entity': 'hour', 'kind': 'container', 'pos': [20, 0, 0], 'at': '2026-11-17T17:00:00.5Z'},
                 {'entity': 'weeks', 'kind': 'container', 'pos': [22, 0, 0], 'at': '2026-11-03T07:04:52Z'},
                 {'entity': 'undated', 'kind': 'container', 'pos': [24, 0, 0]}]
        seen = [{'id': 'live', 'kind': 'container', 'pos': [9, 0, 0]}]
        import datetime
        now = datetime.datetime(2026, 11, 17, 18, 0, tzinfo=datetime.timezone.utc)
        found = census.candidates([], marks, seen, [0, 0, 0], 40, now)
        self.assertEqual({c['entity']: c['sighted'] for c in found},
                         {'old': False, 'today': True, 'live': True, 'hour': True, 'weeks': False, 'undated': False})
        self.assertEqual([c['entity'] for c in census.order(found, [0, 0, 0])],
                         ['today', 'live', 'hour', 'weeks', 'undated', 'old'])
        results = [{'entity': 'old', 'ids': ['old'], 'status': 'rejected', 'bodyDistance': 4.0,
                    'reason': 'invalid_target no storage chest at that entity'},
                   {'entity': 'live', 'ids': ['live'], 'status': 'succeeded', 'openedBefore': False}]
        summary = census.summarise(results, [{'entity': 'live', 'items': []}])
        self.assertEqual((summary['gone'], summary['refused']), (['old'], {}))
        self.assertIn('no longer there (1): old', census.render(summary))
        # Asked from beyond loading range, the same answer proves nothing.
        far = dict(results[0], bodyDistance=120.0)
        self.assertFalse(census.is_gone(far))
        self.assertFalse(census.is_gone(dict(results[0], bodyDistance=None)))
        self.assertEqual(census.summarise([far], [])['gone'], [])

    def test_gone_needs_every_request_to_have_been_in_range(self):
        self.assertIsNone(census.farthest([], [0, 0, 0]))
        self.assertEqual(census.farthest([[3, 0, 0], [50, 0, 0]], [0, 0, 0]), 50)
        unread = census.farthest([[3, 0, 0], [float('inf')] * 3], [0, 0, 0])
        self.assertFalse(census.is_gone({'status': 'rejected', 'reason': 'invalid_target x', 'bodyDistance': unread}))

    def test_an_opened_container_out_of_sight_is_not_ranked_as_probably_gone(self):
        remembered = [{'entity': 'r', 'kind': 'container', 'pos': [30, 0, 0], 'ageS': 14 * 86400, 'items': []}]
        self.assertTrue(census.candidates(remembered, [], [], [0, 0, 0], 40)[0]['sighted'])

    def test_stale_only_keeps_unopened_and_old_chests(self):
        chests = [{'entity': 'new', 'opened': True, 'ageS': 60}, {'entity': 'old', 'opened': True, 'ageS': 5000},
                  {'entity': 'never', 'opened': False, 'ageS': None}]
        self.assertEqual([c['entity'] for c in census.stale_only(chests, 900)], ['old', 'never'])

    def test_next_step_tries_other_ids_then_verified_spots_then_one_step_back(self):
        gone = {'status': 'rejected', 'reason': 'invalid_target no storage chest at that entity'}
        self.assertEqual(census.next_step(gone, ['a'], ['a', 'b'], [], False), ('id', 'b'))
        self.assertIsNone(census.next_step(gone, ['a', 'b'], ['a', 'b'], [], False))
        behind = {'status': 'failed', 'reason': 'unreachable inspection:{"detail":"no position"}',
                  'standing': [[1, 0, 1], [2, 0, 2], [3, 0, 3]]}
        self.assertEqual(census.next_step(behind, ['a'], ['a'], [], False), ('walk', [1, 0, 1]))
        self.assertEqual(census.next_step(behind, ['a'], ['a'], [[1, 0, 1]], False), ('walk', [2, 0, 2]))
        # Two ordinary walks per chest, and no step back after walking.
        self.assertIsNone(census.next_step(behind, ['a'], ['a'], [[1, 0, 1], [2, 0, 2]], False))
        wedged = {'status': 'failed', 'reason': 'unreachable inspection:{"detail":"no position"}', 'standing': []}
        self.assertEqual(census.next_step(wedged, ['a'], ['a'], [], False), ('step_back', None))
        self.assertIsNone(census.next_step(wedged, ['a'], ['a'], [], True))
        self.assertIsNone(census.next_step({'status': 'succeeded'}, ['a'], ['a', 'b'], [], False))
        self.assertIsNone(census.next_step({'status': 'failed', 'reason': 'busy in use'}, ['a'], ['a'], [], False))

    def test_standing_spots_come_from_the_approach_trace_in_order(self):
        def trace(msg, to):
            return {'frame': {'type': 'trace', 'msg': msg, 'data': {'to': to}}}
        records = [trace('approaching verified interaction position', [1, 0, 1]),
                   trace('trying another interaction approach', [9, 9, 9]),
                   trace('approaching verified interaction position', [2, 0, 2]),
                   trace('approaching verified interaction position', [1, 0, 1]),
                   {'frame': {'type': 'task', 'status': 'failed'}}, {'kind': 'log'}]
        self.assertEqual(census.standing_spots(records), [[1, 0, 1], [2, 0, 2]])

    def test_controller_run_is_finished_only_after_its_own_run_ends(self):
        self.assertFalse(census.finished(2, None))
        self.assertFalse(census.finished(2, {'runs': 2, 'running': False}))
        self.assertFalse(census.finished(2, {'runs': 3, 'running': True}))
        self.assertFalse(census.finished(2, {'runs': 3, 'pending': True}))
        self.assertTrue(census.finished(2, {'runs': 3, 'running': False, 'last': 'Census: opened 4'}))
        got = census.census_status([{'name': 'other'}, {'name': census.CENSUS_INSTANCE, 'runs': 1}])
        self.assertEqual(got['runs'], 1)
        definition = census.census_definition('w', [1, 2, 3], 30, 0)
        self.assertEqual(definition['params'], {'world': 'w', 'center': [1, 2, 3], 'radius': 30})
        self.assertEqual((definition['enabled'], definition['trigger']['kind']), (False, 'manual'))
        self.assertEqual(census.census_definition('w', [1, 2, 3], 30, 5)['params']['maxContainers'], 5)

    def test_inventory_counts_only_fresh_containers_in_range(self):
        containers = [
            {'entity': 'a', 'pos': [1, 0, 0], 'ageS': 30, 'items': [{'prefab': 'Wood', 'stack': 50}]},
            {'entity': 'b', 'pos': [2, 0, 0], 'ageS': 40, 'items': [{'prefab': 'Wood', 'stack': 20},
                                                                  {'prefab': 'Stone', 'stack': 3}]},
            {'entity': 'c', 'pos': [3, 0, 0], 'ageS': 50, 'items': []},
            {'entity': 'old', 'pos': [4, 0, 0], 'ageS': 99999, 'items': [{'prefab': 'Wood', 'stack': 999}]},
            {'entity': 'far', 'pos': [400, 0, 0], 'ageS': 30, 'items': [{'prefab': 'Wood', 'stack': 999}]}]
        inv = census.inventory(containers, [0, 0, 0], 30, 600)
        self.assertEqual((inv['fresh'], inv['known'], inv['empty']), (3, 4, 1))
        self.assertEqual(inv['items'], {'Wood': {'count': 70, 'containers': 2}, 'Stone': {'count': 3, 'containers': 1}})
        self.assertIn('Wood: 70 in 2 containers', census.render_inventory(inv))

    def test_summary_counts_refusals_and_items_split_across_chests(self):
        results = [
            {'entity': 'a', 'ids': ['a'], 'status': 'succeeded', 'openedBefore': True},
            {'entity': 'b2', 'ids': ['b', 'b2'], 'status': 'succeeded', 'openedBefore': False},
            {'entity': 'c', 'ids': ['c'], 'status': 'succeeded', 'openedBefore': False},
            {'entity': 'd', 'ids': ['d'], 'status': 'failed',
             'reason': 'unreachable inspection:{"travel":0.4,"trail":[[1,2,3]],"detail":"no clear position"}'},
            {'entity': 'e', 'ids': ['e'], 'status': 'failed',
             'reason': 'unreachable inspection:{"travel":3.1,"trail":[[4,5,6]],"detail":"no clear position"}'},
        ]
        after = [{'entity': 'a', 'items': [{'prefab': 'Wood', 'stack': 50}, {'prefab': 'Stone', 'stack': 3}]},
                 {'entity': 'b', 'items': [{'prefab': 'Wood', 'stack': 20}]}, {'entity': 'c', 'items': []}]
        summary = census.summarise(results, after)
        self.assertEqual((summary['attempted'], summary['opened'], summary['newlyOpened'], summary['empty']),
                         (5, 3, 2, 1))
        # Receipts differ in travel and trail; the same cause is one line.
        self.assertEqual(summary['refused'], {'failed: unreachable: no clear position': ['d', 'e']})
        self.assertEqual(list(summary['items']), ['Wood', 'Stone'])
        self.assertEqual(summary['items']['Wood'], {'count': 70, 'chests': ['a', 'b2']})
        text = census.render(summary)
        self.assertIn('3/5 chests opened', text)
        self.assertIn('Wood: 70 in 2 chests', text)


if __name__ == '__main__':
    unittest.main()
