#!/usr/bin/env python3
"""A successful trial may stay online only with positive safe-home evidence."""
import importlib.util
from pathlib import Path

spec = importlib.util.spec_from_file_location('food_check', Path(__file__).with_name('food-check.py'))
food = importlib.util.module_from_spec(spec); spec.loader.exec_module(food)
s = dict(connected=True, phase='in_world', hp=40, hpMax=40, pos=[0, 30, 0], world='one', connection=2,
         flags=['in_shelter'], home=dict(world='one', pos=[0, 30, 0]))
m = dict(world='one', connection=2, perception=dict(characters=[]))
assert food.safe_to_remain(s, m)
for patch in (dict(hp=20), dict(task=dict(id='new-owner')), dict(flags=[]), dict(pos=[9, 30, 0]), dict(connected=False), dict(home={})):
    assert not food.safe_to_remain(s | patch, m)
for patch in (dict(connection=3), dict(world='other'), dict(perception=None), dict(perception={}),
              dict(perception=dict(characters=[dict(hpFrac=1, hostile=True, pos=[4, 30, 0])]))):
    assert not food.safe_to_remain(s, m | patch)
assert food.safe(s | dict(hp=39), s)
assert food.safe(s | dict(connection=3), s)
print('food supervision, injury/scope and safe-online cleanup passed')
