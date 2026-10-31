#!/usr/bin/env python3
"""Safety policy checks for the bounded marker-food live supervisor."""
import importlib.util
from pathlib import Path
import unittest

spec = importlib.util.spec_from_file_location('marker_food_check', Path(__file__).with_name('marker-food-check.py'))
m = importlib.util.module_from_spec(spec)
spec.loader.exec_module(m)


class Safety(unittest.TestCase):
    def test_http_and_powershell_array_envelopes(self):
        expected = [dict(name='job', runs=1)]
        self.assertEqual(m.rows(expected), expected)
        self.assertEqual(m.rows(dict(value=expected, Count=1)), expected)

    def status(self):
        return dict(connected=True, phase='in_world', world='one', connection=1, hp=70, hpMax=70,
                    pos=[0, 0, 0], flags=['in_shelter'], injurySeq=0, home=dict(world='one', pos=[0, 0, 0]))

    def test_long_travel_requires_a_current_active_meal(self):
        self.assertFalse(m.travel_ready(self.status()))
        self.assertFalse(m.travel_ready(dict(self.status(), food=dict(active=[dict(remainingS=299)]))))
        self.assertTrue(m.travel_ready(dict(self.status(), food=dict(active=[dict(remainingS=300)]))))

    def test_decay_is_not_injury(self):
        before = self.status()
        after = dict(before, hp=65, hpMax=65)
        self.assertIsNone(m.unsafe(after, before))
        self.assertIsNone(m.unsafe(dict(before, hpMax=100), before))
        self.assertIsNotNone(m.unsafe(dict(before, hpMax=100)))
        self.assertIsNotNone(m.unsafe(dict(after, injurySeq=1), before))
        self.assertIsNotNone(m.unsafe(dict(after, connection=2), before))

    def test_remain_requires_scoped_shelter_and_observed_clear_route(self):
        status = self.status()
        mapped = dict(world='one', connection=1, perception=dict(characters=[]))
        self.assertTrue(m.safe_home(status, mapped))
        for change in (dict(perception=None), dict(connection=2), dict(perception=dict(characters=[dict(hpFrac=1, hostile=True, pos=[1, 0, 0])]))):
            self.assertFalse(m.safe_home(status, dict(mapped, **change)))
        self.assertFalse(m.safe_home(dict(status, task=dict(id='owner')), mapped))
        self.assertFalse(m.safe_home(dict(status, flags=[]), mapped))
        self.assertFalse(m.safe_home(dict(status, pos=[0, -3, 0]), mapped))


if __name__ == '__main__':
    unittest.main()
