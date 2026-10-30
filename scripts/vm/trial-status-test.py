"""Progress is observation-based, scoped and never implies objective success."""
import datetime
import importlib.util
import json
from pathlib import Path
import tempfile
import unittest

spec = importlib.util.spec_from_file_location('progress', Path(__file__).with_name('trial-status.py'))
progress = importlib.util.module_from_spec(spec)
spec.loader.exec_module(progress)


class Progress(unittest.TestCase):
    def test_preparation_partial_append_scoped_travel_and_explicit_outcome(self):
        with tempfile.TemporaryDirectory() as name:
            folder = Path(name)
            now = datetime.datetime(2026, 10, 30, tzinfo=datetime.timezone.utc)
            self.assertEqual(progress.report(folder, now)['outcome'], 'preparing')
            rows = []
            for second, connection, x in [(0, 1, 0), (10, 1, 3), (20, 2, 100), (30, 2, 104)]:
                rows.append(json.dumps(dict(at=(now + datetime.timedelta(seconds=second)).isoformat(),
                    status=dict(phase='in_world', world='one', connection=connection, pos=[x, 0, 0],
                                hp=25, hpMax=25, food=dict(active=[])))))
            (folder / 'observations.jsonl').write_text('\n'.join(rows) + '\n{"at":')
            report = progress.report(folder, now + datetime.timedelta(seconds=35))
            self.assertEqual(report['observedTravelM'], 7)
            self.assertEqual(report['observationAgeS'], 5)
            self.assertEqual(report['elapsedS'], 30)
            self.assertEqual(report['outcome'], 'in_progress')
            (folder / 'summary.json').write_text(json.dumps(dict(outcome='aborted', reason='blocked')))
            self.assertEqual(progress.report(folder, now)['outcome'], 'aborted')
            (folder / 'summary.json').write_text(json.dumps(dict(outcome='succeeded', supplied=dict(Honey=4))))
            self.assertEqual(progress.report(folder, now)['supplied'], dict(Honey=4))


if __name__ == '__main__':
    unittest.main()
