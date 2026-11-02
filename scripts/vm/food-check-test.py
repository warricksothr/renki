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
for patch in (dict(hp=20), dict(task=dict(id='new-owner')), dict(flags=[]), dict(pos=[9, 30, 0]), dict(pos=[0, 26, 0]), dict(connected=False), dict(home={})):
    assert not food.safe_to_remain(s | patch, m)
for patch in (dict(connection=3), dict(world='other'), dict(perception=None), dict(perception={}),
              dict(perception=dict(characters=[dict(hpFrac=1, hostile=True, pos=[4, 30, 0])]))):
    assert not food.safe_to_remain(s, m | patch)
assert food.safe(s | dict(hp=39), s)
assert food.safe(s | dict(connection=3), s)

trip = dict(id='goal', kind='food_resupply', status='running', world='one',
            food=dict(injury=0, travelBudget=dict(injury=0, damage=0, hp=40)),
            pending=dict(run='current', taskId='travel', template='food_return',
                         params=dict(dry=True, avoidHostiles=True, recoverThreats=True)))
walking = s | dict(injurySeq=0, injury=dict(seq=0, damage=0, complete=True),
                   journey=dict(id='travel', task='move_to', mode='defend', attempts=1),
                   task=dict(id='travel', task='move_to', status='running'))
hurt = walking | dict(hp=35, injurySeq=111, injury=dict(seq=111, damage=5, complete=True))
assert food.safe(hurt, walking, trip) is None
assert food.safe(hurt | dict(injurySeq=900, injury=dict(seq=900, damage=10, complete=True)), walking, trip) is None
for patch in (dict(injury=dict(seq=111, damage=11, complete=True)), dict(injurySeq=-1), dict(injurySeq=None), dict(hp=29),
              dict(injury=dict(seq=111, damage=5, complete=False)), dict(injury={}),
              dict(task=dict(id='human', task='move_to', status='running')),
              dict(task=dict(id='travel', task='interact', status='running')), dict(connection=3), dict(world='other')):
    assert food.safe(hurt | patch, walking, trip)
for patch in (dict(status='needs_plan'), dict(world='other'), dict(kind='combat_cycle'),
              dict(food={}), dict(pending={}), dict(pending=trip['pending'] | dict(run='')),
              dict(pending=trip['pending'] | dict(params=dict(dry=True, avoidHostiles=True)))):
    assert food.safe(hurt, walking, trip | patch)
finished = trip | dict(status='succeeded', pending=None, food=trip['food'] | dict(injury=111),
                       history=[dict(run='current', taskId='travel', template='food_return', status='succeeded')])
resting = hurt | dict(task=None)
assert food.safe(resting, walking, finished, trip) is None
for patch in (dict(history=[]), dict(history=[dict(run='old', taskId='travel', template='food_return', status='succeeded')]),
              dict(food=trip['food']), dict(id='other-goal')):
    assert food.safe(resting, walking, finished | patch, trip)
assert food.safe(resting | dict(task=dict(id='human', task='follow')), walking, finished, trip)
assert food.safe(resting, walking, finished)  # No previous owned intent.

# Collection owns the same finite allowance only for the exact source/use.
for template, task_name in (('forage', 'interact'), ('pickup_food', 'pickup')):
    pending = dict(run='current', taskId='collection', template=template, entity='1:2',
                   params=dict(entity='1:2', recoverThreats=True))
    collecting = trip | dict(pending=pending)
    before = walking | dict(task=dict(id='collection', task=task_name, status='running'),
                            journey=dict(id='collection', task=task_name, mode='escape', attempts=1))
    after = before | dict(hp=35, injurySeq=111, injury=dict(seq=111, damage=5, complete=True))
    assert food.safe(after, before, collecting) is None
    for patch in (dict(journey=dict(id='collection', task='move_to', mode='escape', attempts=1)),
                  dict(journey=dict(id='stale', task=task_name, mode='escape', attempts=1)),
                  dict(task=dict(id='collection', task='harvest', status='running'))):
        assert food.safe(after | patch, before, collecting)
    for patch in (dict(entity='other'), dict(params=pending['params'] | dict(entity='other')),
                  dict(params=pending['params'] | dict(recoverThreats=False))):
        assert food.safe(after, before, collecting | dict(pending=pending | patch))
    if template == 'forage':
        for mode in ('hold', 'alt', 'repair', 'sleep', 'cook'):
            assert food.safe(after, before, collecting | dict(pending=pending | dict(params=pending['params'] | {mode: True})))
    completed = collecting | dict(pending=None, food=trip['food'] | dict(injury=111),
                                  history=[dict(run='current', taskId='collection', template=template, status='succeeded')])
    assert food.safe(after | dict(task=None), before, completed, collecting) is None
    assert food.safe(after | dict(task=None), before, completed | dict(history=[]), collecting)

sampled = dict(task=dict(id='owned-previous'))
advanced = dict(pending=dict(taskId='owned-next'), history=[dict(taskId='owned-previous', status='succeeded')])
assert food.goal_owned_task(sampled, advanced)
assert food.goal_owned_task(dict(task=dict(id='owned-next')), advanced)
assert not food.goal_owned_task(dict(task=dict(id='new-owner')), advanced)
assert not food.goal_owned_task(sampled, dict(pending=dict(taskId='owned-next'), history=[]))
assert not food.goal_owned_task(sampled, dict(history=[dict(taskId='owned-previous', status='running')]))

shortage = 'household source markers exhausted or unsafe; wait for regrowth or revise sources'
summary = dict(outcome='failed', status='needs_plan', reason=shortage)
returned = dict(status='needs_plan', reason=shortage, history=[dict(template='food_return', status='succeeded')])
assert food.completed_or_returned_shortage(summary, returned)
assert summary['outcome'] == 'failed'
for patch in (dict(status='running'), dict(reason='injury'), dict(pending=dict(taskId='return')),
              dict(history=[]), dict(history=[dict(template='food_return', status='failed')]),
              dict(history=[dict(template='search_food', status='succeeded')])):
    assert not food.completed_or_returned_shortage(summary, returned | patch)
assert not food.completed_or_returned_shortage(summary | dict(reason='VM request failed'), returned)
assert not food.completed_or_returned_shortage(summary, None)
print('food supervision, injury/scope and safe-online cleanup passed')

intent = dict(run='current', taskId='escort-task')
active = dict(task=dict(id='escort-task', task='escort'), escort=dict(id='escort-task', leader='Nimah', mode='wait'))
assert food.escort_active(active, intent, 'Nimah')
for patch in (dict(task=dict(id='owner', task='escort')), dict(escort=dict(id='escort-task', leader='other', mode='wait')),
              dict(escort=dict(id='escort-task', leader='Nimah', mode='escape'))):
    assert not food.escort_active(active | patch, intent, 'Nimah')
history = [dict(frame=dict(type='task', id='escort-task', task='escort', args=dict(player='Nimah', defend=False, collect=False))),
           dict(frame=dict(type='task', re='escort-task', status='accepted')),
           dict(frame=dict(type='task', re='escort-task', status='running'))]
assert food.escort_receipt(history, intent, 'Nimah')
assert not food.escort_receipt(history[:-1], intent, 'Nimah')
assert not food.escort_receipt(history, dict(taskId='escort-task'), 'Nimah')
assert not food.escort_receipt(history, intent, 'other')
assert not food.escort_receipt(history, dict(run='old', taskId='old-task'), 'Nimah')


definition = food.household_definition(dict(world='one', stock=dict(targets=dict(Mushroom=4))), 'storage')
assert definition['trigger']['marker'] == dict(world='one', name='storage', kind='storage')
assert definition['trigger']['dwellS'] == 30
assert definition['repeat'] == dict(mode='trigger', cooldownS=300, maxRuns=1)
print('household idle episode uses the admitted finite repeat policy')
