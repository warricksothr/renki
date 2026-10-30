import copy
import json
from pathlib import Path
import tempfile
import unittest

import gather_trial as gather


class GatherEvidence(unittest.TestCase):
    def test_controller_can_approach_a_distant_observed_site_without_relaxing_protection(self):
        status = dict(pos=[0, 30, 0])
        tree = dict(id='tree', kind='tree', prefab='Beech1', minTool=0, pos=[30, 30, 0])
        perception = dict(resources=[tree], structures=[], characters=[])
        target, diagnostic = gather.select_site(status, perception)
        self.assertEqual(target, tree)
        self.assertEqual(diagnostic['nearest'][0]['distanceM'], 30)
        perception['characters'] = [dict(pos=[40, 30, 0], hpFrac=1)]
        target, diagnostic = gather.select_site(status, perception)
        self.assertIsNone(target)
        self.assertFalse(diagnostic['nearest'][0]['worksiteClear'])
        perception['characters'] = []
        tree['pos'] = [61, 30, 0]
        self.assertIsNone(gather.select_site(status, perception)[0])

    def fixture(self):
        summary = dict(outcome='awaiting_proof', behaviourName='trial', connection=7, world='world', worldUid='uid',
                       before={'Wood': 22}, after={'Wood': 22}, beforeInventorySeq=10, afterInventorySeq=70,
                       site=[5, 30, 0], finalPos=[5, 30, 1], to=['chest'], tree={'id': 'tree'}, initialResourceIDs=['tree', 'old'])
        rows = []
        def frame(**f):
            rows.append(dict(conn=7, frame=f))
        frame(type='perception', worldUid='uid', resources=[
            dict(id='tree', kind='tree', prefab='Beech1'), dict(id='a', kind='log', prefab='beech_log'),
            dict(id='b', kind='log', prefab='beech_log_half')])
        for i, (key, task, args) in enumerate([
            ('h1', 'harvest', dict(entity='tree')), ('h2', 'harvest', dict(entity='a')),
            ('h3', 'harvest', dict(entity='b')), ('d1', 'deposit', dict(container='chest')),
            ('r1', 'move_to', dict(to=summary['site']))]):
            rows.append(dict(msg='behaviour step', attrs=dict(instance='trial', id=key, task=task, step=i+1)))
            if task == 'deposit':
                frame(type='inventory', seq=45, items=[dict(prefab='Wood', stack=42)])
            frame(type='task', id=key, task=task, args=args)
            if task == 'harvest':
                frame(type='trace', task=key, source='harvest', msg='resource broke after local hit', data=dict(entity=args['entity'], hits=5))
            if task == 'deposit':
                frame(type='container', task=key, direction='deposit', entity='chest', moved={'Wood': 20}, seq=50)
            frame(type='task', re=key, status='succeeded')
        return summary, rows

    def verify(self, summary, rows):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory)/'audit.jsonl'
            path.write_text(''.join(json.dumps(r)+'\n' for r in rows))
            return gather.verify(path, summary)

    def test_requires_controller_attributed_tree_logs_gains_receipt_and_return(self):
        summary, rows = self.fixture()
        result = self.verify(summary, rows)
        self.assertEqual(result['outcome'], 'succeeded')
        self.assertEqual(result['logsHarvested'], 2)
        self.assertEqual(result['deposited'], {'Wood': 20})
        self.assertTrue(result['returned'])
        self.assertEqual(len(result['tasks']), 5)

    def test_incomplete_or_substituted_evidence_never_earns_success(self):
        summary, original = self.fixture()
        for kind in ('no_break', 'wrong_break_entity', 'foreign_receipt', 'wrong_chest', 'lost_baseline',
                     'preexisting_log', 'no_return', 'stale_inventory', 'wrong_scope', 'duplicate_receipt', 'lost_gain',
                     'substituted_tree', 'duplicate_log', 'remaining_gain', 'boolean_count'):
            s, rows = copy.deepcopy(summary), copy.deepcopy(original)
            receipt = next(r['frame'] for r in rows if r.get('frame', {}).get('type') == 'container')
            if kind == 'no_break': rows = [r for r in rows if r.get('frame', {}).get('type') != 'trace']
            if kind == 'wrong_break_entity': next(r['frame']['data'] for r in rows if r.get('frame', {}).get('type') == 'trace')['entity'] = 'substitute'
            if kind == 'foreign_receipt': receipt['task'] = 'other'
            if kind == 'wrong_chest': receipt['entity'] = 'other'
            if kind == 'lost_baseline': s['after']['Wood'] = 21
            if kind == 'preexisting_log': s['initialResourceIDs'].append('a')
            if kind == 'no_return': s['finalPos'] = [50, 30, 0]
            if kind == 'stale_inventory': s['afterInventorySeq'] = 49
            if kind == 'wrong_scope': rows[0]['frame']['worldUid'] = 'other'
            if kind == 'duplicate_receipt': rows.append(dict(conn=7, frame=copy.deepcopy(receipt)))
            if kind == 'lost_gain': receipt['moved']['Wood'] = 19
            if kind == 'substituted_tree': s['tree']['id'] = 'another'
            if kind == 'duplicate_log': next(r['frame'] for r in rows if r.get('frame', {}).get('id') == 'h3')['args']['entity'] = 'a'
            if kind == 'remaining_gain': s['after']['Wood'] = 23
            if kind == 'boolean_count': receipt['moved']['Feathers'] = False
            with self.subTest(kind=kind), self.assertRaises(RuntimeError):
                self.verify(s, rows)

    def test_failures_keep_step_receipts_without_inventing_success(self):
        summary, rows = self.fixture()
        summary['outcome'] = 'failed'
        result = self.verify(summary, rows)
        self.assertNotIn('outcome', result)
        self.assertEqual(len(result['tasks']), 5)

    def test_chests_are_hints_reconciled_only_to_unique_live_identity(self):
        status = dict(world='world', pos=[0, 30, 0], home=dict(world='world', pos=[0, 30, 0]))
        mapped = dict(containers=[dict(entity='old', pos=[1, 30, 0])], perception=dict(structures=[dict(kind='container', id='new', pos=[1, 30, 0])]))
        self.assertEqual(gather.destinations(status, mapped), ['new'])
        mapped['perception']['structures'].append(dict(kind='container', id='ambiguous', pos=[1, 30, 0]))
        with self.assertRaisesRegex(RuntimeError, 'no remembered'):
            gather.destinations(status, mapped)


if __name__ == '__main__':
    unittest.main()
