#!/usr/bin/env python3
import importlib.util
from pathlib import Path
spec = importlib.util.spec_from_file_location('escort', Path(__file__).with_name('escort-check.py'))
escort = importlib.util.module_from_spec(spec)
spec.loader.exec_module(escort)
status = dict(connected=True, phase='in_world', world='one', hp=40, hpMax=50,
              owners=['Nimah'], players={'profiles': {'Pilot': 'operator', 'Friend': 'companion', 'Guest': 'guest'}})
assert escort.unsafe(status, 'one') is None
for patch in [dict(connected=False), dict(phase='dead'), dict(world='two'), dict(hp=14), dict(hp=19), dict(hp=float('nan'))]:
    assert escort.unsafe(dict(status, **patch), 'one')
assert escort.protected(status, 'Nimah') == ['Nimah', 'Pilot']
try:
    escort.protected(status, 'Friend')
    raise AssertionError('companion received operator protection')
except RuntimeError:
    pass
print('escort supervision policies passed')
