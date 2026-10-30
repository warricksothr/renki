import importlib.util
from pathlib import Path
import unittest

spec = importlib.util.spec_from_file_location('marker', Path(__file__).with_name('marker-check.py'))
marker = importlib.util.module_from_spec(spec)
spec.loader.exec_module(marker)


class Marker(unittest.TestCase):
    def test_reports_both_native_states_and_restores_on_world_change(self):
        for fail in (False, True):
            with self.subTest(fail=fail):
                state = dict(connected=True, phase='in_world', world='one', connection=1, character='Tas', hp=25, hpMax=25, sharePosition=False, presence=dict(sharePosition=False))
                changes = []
                def call(method, path, body=None):
                    if method == 'POST':
                        state['sharePosition'] = body['value']
                        changes.append(body['value'])
                        if fail:
                            state['world'] = 'other'
                        return {}
                    return state
                if fail:
                    with self.assertRaisesRegex(RuntimeError, 'world changed'):
                        marker.verify(call, lambda _: None, 'on')
                    self.assertEqual(changes, [True, False])
                else:
                    summary = marker.verify(call, lambda _: None, 'restore')
                    self.assertEqual(changes, [True, False, False])
                    self.assertEqual([r['observed'] for r in summary['transitions']], changes)
                    self.assertFalse(summary['finalState'])


if __name__ == '__main__':
    unittest.main()
