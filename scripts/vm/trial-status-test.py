"""Progress is observation-based, scoped and never implies objective success."""
import datetime
import contextlib
import importlib.util
import io
import json
from pathlib import Path
import tempfile
import unittest
from unittest import mock

spec = importlib.util.spec_from_file_location('progress', Path(__file__).with_name('trial-status.py'))
progress = importlib.util.module_from_spec(spec)
spec.loader.exec_module(progress)


class Progress(unittest.TestCase):
    def test_goal_range_uses_observed_positions_in_the_matching_running_world(self):
        with tempfile.TemporaryDirectory() as name:
            folder = Path(name)
            now = datetime.datetime(2026, 10, 30, tzinfo=datetime.timezone.utc)
            goal = dict(id='goal', world='one', status='running', center=[100, 0, 0])
            rows = []
            for second, world, status, x in [(0, 'one', 'running', 100), (1, 'one', 'running', 170),
                                              (2, 'one', 'needs_plan', 250), (3, 'two', 'running', 300)]:
                rows.append(json.dumps(dict(at=(now + datetime.timedelta(seconds=second)).isoformat(),
                    status=dict(phase='in_world', world=world, pos=[x, 0, 0], goals=[dict(goal, status=status)]))))
            (folder / 'observations.jsonl').write_text('\n'.join(rows))
            (folder / 'goal.json').write_text(json.dumps(dict(goal, status='needs_plan')))
            self.assertEqual(progress.report(folder, now)['goal']['maxObservedRangeM'], 70)

    def test_goal_progress_shows_attempted_bearings_without_policy_or_perception(self):
        with tempfile.TemporaryDirectory() as name:
            folder = Path(name)
            goal = dict(status='running', attempts=2, maxAttempts=6, policy='large source',
                        pending=dict(search=dict(sector=4, center=[-80, 30, 0], radius=24)),
                        searches=[dict(plan=dict(sector=0, origin=[0, 0, 0], center=[80, 30, 0], radius=24), empty=True,
                                       outcome=dict(reason='invalid_target', detail='long detail'))])
            (folder / 'goal.json').write_text(json.dumps(goal))
            result = progress.report(folder)['goal']
            self.assertEqual(result['pendingSector'], 4)
            self.assertEqual(result['pendingOffset'], 0)
            self.assertEqual(result['pendingSearchCenter'], [-80, 30, 0])
            self.assertEqual(result['searches'], [dict(sector=0, offset=0, center=[80, 30, 0], radius=24, empty=True, reason='invalid_target')])
            self.assertNotIn('policy', result)
            (folder / 'summary.json').write_text(json.dumps(dict(outcome='needs_plan',
                goal=dict(goal, status='needs_plan', attempts=6, pending=None))))
            result = progress.report(folder)['goal']
            self.assertEqual(result['status'], 'needs_plan')
            self.assertEqual(result['attempts'], 6)
            self.assertIsNone(result['pendingSector'])
            self.assertIsNone(result['pendingOffset'])
            (folder / 'summary.json').unlink()
            (folder / 'goal.json').write_text('{')
            self.assertNotIn('goal', progress.report(folder))

    def test_kill_and_return_proofs_remain_distinct_in_progress(self):
        with tempfile.TemporaryDirectory() as name:
            folder = Path(name)
            goal = dict(status='needs_plan', proof=dict(taskId='fight', entity='grey', detail='observed Greyling death', unused='large'),
                        postCombat=dict(phase='confirm', receipt=dict(taskId='return', status='failed', reason='unreachable')))
            (folder / 'goal.json').write_text(json.dumps(goal))
            result = progress.report(folder)['goal']
            self.assertEqual(result['killProof']['taskId'], 'fight')
            self.assertEqual(result['postCombat']['receipt']['taskId'], 'return')
            self.assertEqual(result['status'], 'needs_plan')
            self.assertNotIn('unused', result['killProof'])

    def test_latest_gather_run_includes_delivery_and_missing_terminal_result(self):
        with tempfile.TemporaryDirectory() as name:
            root = Path(name)
            (root / '.checks' / 'combat-trial-20261030T100000Z').mkdir(parents=True)
            folder = root / '.checks' / 'gather-trial-20261030T110000Z'
            folder.mkdir()
            (folder / 'summary.json').write_text(json.dumps(dict(outcome='failed', deposited={'Wood': 20},
                returned=False, tasks=[dict(id='h1', task='harvest', result=None)])))
            with mock.patch.object(progress, 'ROOT', root), mock.patch.object(progress.sys, 'argv', ['trial-status']), contextlib.redirect_stdout(io.StringIO()) as output:
                progress.main()
            result = json.loads(output.getvalue())
            self.assertEqual(result['evidence'], str(folder))
            self.assertEqual(result['deposited'], {'Wood': 20})
            self.assertFalse(result['returned'])
            self.assertIsNone(result['tasks'][0]['status'])

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

    def test_combat_failure_is_visible_without_opening_raw_summary(self):
        with tempfile.TemporaryDirectory() as name:
            folder = Path(name)
            native = dict(status='failed', reason='refused', detail='cannot return along a checked scouting route',
                          seq=777, re='t-1')
            (folder / 'summary.json').write_text(json.dumps(dict(outcome='failed', result=native)))
            result = progress.report(folder)
            self.assertEqual(result['result'], {k: native[k] for k in ('status', 'reason', 'detail')})
            self.assertEqual(result['outcome'], 'failed')
            self.assertEqual(result['tasks'], [])


if __name__ == '__main__':
    unittest.main()
