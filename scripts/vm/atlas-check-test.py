import importlib.util
from pathlib import Path
import unittest

spec = importlib.util.spec_from_file_location('atlas_check', Path(__file__).with_name('atlas-check.py'))
atlas = importlib.util.module_from_spec(spec)
spec.loader.exec_module(atlas)


class AtlasEvidence(unittest.TestCase):
    def test_canonical_borders_must_agree_in_value_and_age(self):
        def tile(x, i, height=40):
            out = dict(x=x, z=0, heights=[None]*25, biomes=[None]*25,
                       observedAt=[None]*25, sources=[None]*25)
            out['heights'][i] = height
            out['biomes'][i] = 'Meadows'
            out['observedAt'][i] = '2026-10-30T16:00:00Z'
            out['sources'][i] = 'bot'
            return out
        self.assertEqual(len(atlas.samples([tile(-1, 4), tile(0, 0)])), 1)
        with self.assertRaisesRegex(RuntimeError, 'Conflicting overlapping'):
            atlas.samples([tile(-1, 4), tile(0, 0, 41)])

    def test_retention_oracle_detects_missing_or_older_samples(self):
        old = {'0,0': [40, 'Meadows', '2026-10-30T16:00:00Z', 'bot']}
        fresh = {'0,0': [41, 'Meadows', '2026-10-30T11:00:01-05:00', 'bot']}
        self.assertEqual(atlas.retained(old, fresh), 1)
        with self.assertRaisesRegex(RuntimeError, 'samples lost'):
            atlas.retained(old, {})
        fresh['0,0'][2] = '2026-10-30T15:59:59Z'
        with self.assertRaisesRegex(RuntimeError, 'regressed'):
            atlas.retained(old, fresh)


if __name__ == '__main__':
    unittest.main()
