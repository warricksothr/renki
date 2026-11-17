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


if __name__ == '__main__':
    unittest.main()
