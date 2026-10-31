#!/usr/bin/env python3
import importlib.util
import json
from pathlib import Path
from resupply_trial import candidates

spec = importlib.util.spec_from_file_location('cooking_check', Path(__file__).with_name('cooking-check.py'))
cooking = importlib.util.module_from_spec(spec); spec.loader.exec_module(cooking)

status = dict(connected=True, phase='in_world', world='one', connection=1, hp=30, hpMax=32,
              injurySeq=0, pos=[0, 30, 0], flags=['grounded'], food={'active': [{'prefab': 'Raspberry', 'remainingS': 400}]},
              home={'world': 'one', 'pos': [0, 30, 0]})
assert cooking.safe(status) is None
for patch in [dict(hp=19), dict(hp=float('nan')), dict(phase='dead'), dict(food={'active': []}), dict(location={'inWater': True})]:
    assert cooking.safe(dict(status, **patch))
assert cooking.safe(dict(status, injurySeq=1), status)
assert cooking.safe(dict(status, hp=28), status)
assert cooking.safe(dict(status, connection=2), status)
# A fresh cooked meal may raise the maximum before regeneration; injury still aborts.
assert cooking.safe(dict(status, hpMax=60), status) is None
unfed = status | dict(food={'active': []})
assert cooking.safe(unfed, require_food=False) is None
assert cooking.safe(unfed)
assert cooking.safe(unfed | dict(hp=19), require_food=False)
assert cooking.safe(unfed | dict(injurySeq=1), status, require_food=False)
assert cooking.safe(unfed | dict(location={'inWater': True}), require_food=False)
assert cooking.home_search_ready(status | dict(pos=[21, 30, 0]))
assert cooking.home_search_ready(status | dict(pos=[32, 30, 0]))
assert not cooking.home_search_ready(status | dict(pos=[32.1, 30, 0]))
assert not cooking.home_search_ready(status | dict(world='other'))
assert not cooking.home_search_ready(status | dict(home={}))

station = dict(id='1:2', kind='station', pos=[2, 30, 0], cooking={'supported': True, 'heated': True,
               'slots': [{'item': '', 'state': 'empty'}]})
mapped = dict(world='one', connection=1, perception={'structures': [station]})
assert cooking.choose_station(status, mapped) == '1:2'
assert cooking.choose_station(status, dict(mapped, connection=2)) is None
for changed in [dict(supported=False), dict(heated=False), dict(slots=[]), dict(slots=[{'item': 'RawMeat', 'state': 'cooking'}])]:
    modified = dict(station, cooking=station['cooking'] | changed)
    assert cooking.choose_station(status, dict(mapped, perception={'structures': [modified]})) is None
assert cooking.choose_station(status, dict(mapped, perception={'structures': [dict(station, pos=[0, 45, 0])]})) is None
cold = dict(station, cooking=station['cooking'] | dict(heated=False))
assert cooking.choose_station(status, dict(mapped, perception={'structures': [cold]}), wood=True) == '1:2'

chest = dict(id='chest', kind='container', pos=[2, 30, 0])
storage = {'perception': {'structures': [chest, dict(chest)]}, 'containers': []}
options = candidates(status, storage, {}, foods=cooking.RAW)
assert len(options) == 1 and options[0][3] == 'RawMeat'
assert cooking.search_chests(status, storage) == {'chest'}
ghost = dict(entity='ghost', pos=[5, 30, 0], items=[dict(prefab='RawMeat', stack=5)])
assert cooking.search_chests(status, dict(storage, containers=[ghost])) == {'chest'}
opened = {'chest': dict(entity='chest', pos=[2, 30, 0], items=[dict(prefab='DeerMeat', stack=8)])}
options = candidates(status, storage, {}, opened, foods=cooking.RAW)
assert len(options) == 1 and options[0][3] == 'DeerMeat'
assert candidates(status, storage, {}, opened) == []  # Ordinary resupply still excludes raw ingredients.
history = [{'frame': dict(type='task', id='t-1', task='withdraw')}, {'frame': dict(type='task', re='t-1', status='running')},
           {'frame': dict(type='task', re='t-2', status='succeeded')},
           {'frame': dict(type='task', re='t-1', status='failed', reason='refused')}]
assert cooking.terminal({'value': history}, 't-1')['status'] == 'failed'
assert cooking.terminal(history, 'missing') is None
native = json.loads((cooking.ROOT / 'internal/protocol/testdata/plugin/task_failed.json').read_text())
assert cooking.terminal([{'frame': native}], native['re'])['status'] == 'failed'

for mode, message in [('direct_spawn', 'normal station output spawned'),
                      ('remote_correlation', 'correlated remote station output observed')]:
    receipt = [dict(msg='normal cooking removal request observed', data=dict(evidence=mode, requestedCount=1)),
               dict(msg=message, data=dict(entity='drop', output='CookedMeat', count=1)),
               dict(msg='verified cooking drop pickup', data=dict(entity='drop', output='CookedMeat', count=1)),
               dict(msg='verified cooked inventory receipt', data=dict(evidence=mode, output='CookedMeat', count=1))]
    assert cooking.delivery_receipt(receipt) == dict(evidence=mode, output='CookedMeat', count=1)
    for missing in range(len(receipt)):
        assert cooking.delivery_receipt(receipt[:missing] + receipt[missing+1:]) is None
    for index, patch in [(0, dict(requestedCount=2)), (1, dict(entity='other')), (2, dict(count=2)),
                         (2, dict(output='Raspberry')), (3, dict(evidence='unknown'))]:
        changed = list(receipt); changed[index] = receipt[index] | dict(data=receipt[index]['data'] | patch)
        assert cooking.delivery_receipt(changed) is None
print('cooking supervision, unique chest candidates and scoped receipts passed')
