#!/usr/bin/env python3
"""Offline checks of the route report."""
import importlib.util
from pathlib import Path
import unittest

spec = importlib.util.spec_from_file_location('routes', Path(__file__).with_name('routes.py'))
routes = importlib.util.module_from_spec(spec)
spec.loader.exec_module(routes)


class RoutesTest(unittest.TestCase):
    def test_offline_and_empty_reports(self):
        self.assertIn('not in a world', routes.render({'world': '', 'stats': {}}))
        text = routes.render({'world': 'w', 'stats': {'cells': 0, 'links': 0, 'levels': [], 'groups': []},
                              'body': {'pos': [1, 2, 3], 'cell': '1,4,3', 'known': False}})
        self.assertIn('0 walked cells', text)
        self.assertIn('nothing walked yet', text)

    def test_report_names_levels_groups_and_the_body(self):
        status = {'world': 'w', 'stats': {
            'cells': 40, 'links': 70, 'walked': 39, 'failed': 1, 'newest': '2026-11-17T19:40:02.123Z',
            'levels': [{'y': 31, 'cells': 30}, {'y': 33, 'cells': 10}],
            'groups': [{'cells': 30, 'centre': [-168.2, 31, -467.9], 'minY': 30.9, 'maxY': 31.2},
                       {'cells': 10, 'centre': [-204, 33, -467], 'minY': 32.9, 'maxY': 33}]},
            'body': {'pos': [-204.3, 32.97, -467.57], 'cell': 'x', 'known': False,
                     'nearest': [-203, 33, -467], 'nearestM': 1.4}}
        text = routes.render(status)
        self.assertIn('40 walked cells, 70 links (39 walked, 31 inferred back, 1 failed)', text)
        self.assertIn('height levels: 31 m (30), 33 m (10)', text)
        self.assertIn('2 separate groups: no walked way between them', text)
        self.assertIn('30 cells around [-168, 31, -468], heights 30.9 to 31.2 m', text)
        self.assertIn('is off walked ground; nearest is [-203.0, 33.0, -467.0], 1.4 m away', text)
        status['body']['known'] = True
        self.assertIn('stands on walked ground', routes.render(status))
        status['stats']['groups'] = status['stats']['groups'][:1]
        self.assertNotIn('separate groups', routes.render(status))

    def test_plan_path_tells_positions_from_place_names(self):
        self.assertEqual(routes.plan_path('-169.69, 31.03, -467.8'), '/api/routes/plan?to=-169.69%2C31.03%2C-467.8')
        self.assertEqual(routes.plan_path('home'), '/api/routes/plan?place=home')
        self.assertEqual(routes.plan_path('new home storage west', '1,2,3'),
                         '/api/routes/plan?place=new%20home%20storage%20west&from=1%2C2%2C3')
        for bad in (('', ''), ('home', 'here')):
            with self.assertRaises(SystemExit):
                routes.plan_path(*bad)

    def test_plan_report_lists_legs_with_climbs_and_unwalked_directions(self):
        plan = {'found': True, 'explain': '3 legs, 14 m over 12 walked cells.', 'place': 'home',
                'start': {'pos': [0, 31, 0]}, 'target': {'pos': [4, 33, 9]},
                'legs': [{'to': [8, 31, 0], 'lengthM': 8, 'climb': 0, 'cells': 8},
                         {'to': [8, 33, 4], 'lengthM': 4.5, 'climb': 2, 'cells': 4, 'inferred': 4},
                         {'to': [4, 33, 9], 'lengthM': 1.5, 'climb': 0, 'cells': 0}]}
        text = routes.render_plan(plan)
        self.assertIn('to home [4.0, 33.0, 9.0]', text)
        self.assertIn(' 1. [8.0, 31.0, 0.0]  8.0 m', text)
        self.assertIn('(up 2.0 m; 4 of 4 links never walked this way)', text)
        self.assertIn('(off walked ground)', text)

    def test_plan_report_says_why_there_is_no_route_and_what_to_walk(self):
        plan = {'found': False, 'reason': 'no_walked_way_between', 'explain': 'Not joined.', 'legs': [],
                'start': {'pos': [0, 31, 0], 'cell': '0,62,0'}, 'target': {'pos': [9, 33, 0], 'cell': '9,66,0'},
                'closest': [4.5, 31, 0.5], 'gapM': 4.9}
        text = routes.render_plan(plan)
        self.assertIn('NO ROUTE (no_walked_way_between): Not joined.', text)
        self.assertIn('walk from [4.5, 31.0, 0.5] toward the target to close the 4.9 m gap', text)
        unknown = {'found': False, 'reason': 'unknown_place', 'place': 'nowhere', 'legs': [],
                   'explain': 'No saved place is named "nowhere". Saved places: home, pantry.',
                   'start': {'pos': [0, 31, 0]}, 'target': {'pos': [0, 0, 0]}}
        self.assertIn("route to 'nowhere': NO ROUTE (unknown_place)", routes.render_plan(unknown))
        self.assertIn('Saved places: home, pantry', routes.render_plan(unknown))
        off = {'found': False, 'reason': 'target_off_walked_ground', 'explain': 'x', 'legs': [],
               'start': {'pos': [0, 31, 0], 'cell': 'a'},
               'target': {'pos': [9, 40, 0], 'cellPos': [8, 33, 0], 'distanceM': 7.1}}
        self.assertIn('nearest walked cell to the target: [8.0, 33.0, 0.0], 7.1 m away', routes.render_plan(off))


if __name__ == '__main__':
    unittest.main()
