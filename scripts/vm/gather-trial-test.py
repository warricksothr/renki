import copy
import json
from pathlib import Path
import tempfile
import unittest

import gather_trial as gather
from harvest_trial import health_loss


class GatherEvidence(unittest.TestCase):
    def test_passive_deer_is_not_a_hostile_trial_approach(self):
        current = dict(pos=[0, 0, 0])
        deer = dict(id='deer', kind='creature', prefab='Deer', hostile=True, hpFrac=1, pos=[8, 0, 0])
        self.assertFalse(gather.nearby_threat(current, dict(characters=[deer])))
        for change in (dict(prefab='Neck'), dict(prefab='Greyling'), dict(targetingMe=True), dict(kind='player')):
            self.assertTrue(gather.nearby_threat(current, dict(characters=[dict(deer, **change)])))
        self.assertFalse(gather.nearby_threat(current, dict(characters=[dict(deer, prefab='Greyling')]), 'deer'))

    def test_defense_proximity_exception_requires_attributed_matching_native_attack(self):
        current = dict(task=dict(id='fight', task='attack'), combat=dict(id='fight', entity='pursuer'))
        rows = [dict(msg='behaviour step', attrs=dict(instance='trial', id='fight')),
                dict(frame=dict(type='task', id='fight', task='attack', args=dict(entity='pursuer', kite=True)))]
        self.assertEqual(gather.defense_target(current, rows, 'trial'), 'pursuer')
        self.assertIsNone(gather.defense_target(current, rows, 'foreign'))
        self.assertIsNone(gather.defense_target(dict(current, combat=dict(id='other', entity='pursuer')), rows, 'trial'))
        self.assertIsNone(gather.defense_target(current, rows[1:], 'trial'))
        for field, value in [('entity', 'substitute'), ('kite', False)]:
            changed = copy.deepcopy(rows)
            changed[1]['frame']['args'][field] = value
            self.assertIsNone(gather.defense_target(current, changed, 'trial'))

    def test_refused_cut_must_defend_and_retry_same_entity_with_exact_break_proof(self):
        summary, rows = self.fixture()
        summary['defend'] = True
        extra = [dict(msg='behaviour step', attrs=dict(instance='trial', id='refused', task='harvest')),
                 dict(conn=7, frame=dict(type='task', id='refused', task='harvest', args=dict(entity='tree'))),
                 dict(conn=7, frame=dict(type='task', re='refused', status='failed', reason='refused')),
                 dict(msg='behaviour step', attrs=dict(instance='trial', id='fight', task='attack')),
                 dict(conn=7, frame=dict(type='task', id='fight', task='attack', args=dict(entity='pursuer', kite=True, style='melee'))),
                 dict(conn=7, frame=dict(type='trace', task='fight', source='combat', msg='death_observed', data=dict(entity='pursuer'))),
                 dict(conn=7, frame=dict(type='task', re='fight', status='succeeded', detail='observed Greyling death'))]
        rows[1:1] = extra
        proof = self.verify(summary, rows)
        self.assertEqual((proof['treesHarvested'], proof['logsHarvested'], proof['defenses']), (1, 2, 1))
        for mode in ('not opted in', 'wrong death', 'disappearance', 'foreign attack', 'wrong retry', 'no defense', 'unrecoverable', 'missing break'):
            s, rr = copy.deepcopy(summary), copy.deepcopy(rows)
            if mode == 'not opted in': s['defend'] = False
            if mode == 'wrong death': next(r['frame'] for r in rr if r.get('frame', {}).get('task') == 'fight')['data']['entity'] = 'other'
            if mode == 'disappearance': next(r['frame'] for r in rr if r.get('frame', {}).get('re') == 'fight')['detail'] = 'enemy disappeared'
            if mode == 'foreign attack': next(r['attrs'] for r in rr if r.get('attrs', {}).get('id') == 'fight')['instance'] = 'foreign'
            if mode == 'wrong retry': next(r['frame'] for r in rr if r.get('frame', {}).get('id') == 'refused')['args']['entity'] = 'other'
            if mode == 'no defense': rr = [r for r in rr if r.get('attrs', {}).get('id') != 'fight']
            if mode == 'unrecoverable': next(r['frame'] for r in rr if r.get('frame', {}).get('re') == 'refused')['reason'] = 'unreachable'
            if mode == 'missing break': rr = [r for r in rr if not (r.get('frame', {}).get('type') == 'trace' and r['frame'].get('task') == 'h1')]
            with self.subTest(mode=mode), self.assertRaises(RuntimeError): self.verify(s, rr)

    def test_food_refresh_needs_unchanged_injury_evidence_and_never_hides_a_hit(self):
        before = dict(hp=60.7, hpMax=60.7, injurySeq=0, food=dict(active=[dict(prefab='CookedDeerMeat', remainingS=603)]))
        after = dict(hp=60, hpMax=66.1, injurySeq=0, food=dict(active=[dict(prefab='CookedDeerMeat', remainingS=1197)]))
        self.assertFalse(health_loss(before, after))
        self.assertTrue(health_loss(before, dict(after, injurySeq=100)))
        self.assertTrue(health_loss(before, dict(after, hp=66.1, injurySeq=100)))
        self.assertTrue(health_loss(before, dict(after, injurySeq=None)))
        self.assertTrue(health_loss(dict(before, injurySeq=None), after))
        self.assertTrue(health_loss(before, dict(after, food=before['food'])))

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

    def test_two_tree_home_trial_requires_two_complete_chains_and_exact_combined_deposit(self):
        summary, rows = self.fixture()
        summary.update(maxTrees=2, returnHome=True, home=[0, 30, -10], finalPos=[0, 30, -9])
        rows[0]['frame']['resources'] += [dict(id='tree2', kind='tree', prefab='Beech1'),
            dict(id='c', kind='log', prefab='beech_log'), dict(id='d', kind='log', prefab='beech_log_half')]
        extra = []
        for key, entity in [('h4', 'tree2'), ('h5', 'c'), ('h6', 'd')]:
            extra += [dict(msg='behaviour step', attrs=dict(instance='trial', id=key, task='harvest')),
                dict(conn=7, frame=dict(type='task', id=key, task='harvest', args=dict(entity=entity))),
                dict(conn=7, frame=dict(type='trace', task=key, source='harvest', msg='resource broke after local hit', data=dict(entity=entity, hits=5))),
                dict(conn=7, frame=dict(type='task', re=key, status='succeeded'))]
        insertion = next(i for i,r in enumerate(rows) if r.get('attrs',{}).get('id') == 'd1')
        rows[insertion:insertion] = extra
        for r in rows:
            f=r.get('frame',{})
            if f.get('type')=='inventory': f['items'][0]['stack']=62
            if f.get('type')=='container': f['moved']['Wood']=40
            if f.get('id')=='r1': f['args']['to']=summary['home']
        result=self.verify(summary,rows)
        self.assertEqual((result['treesHarvested'],result['logsHarvested'],result['woodGain']),(2,4,40))
        for failure in ('same_tree_twice','missing_log_chain','wrong_deposit','under_home'):
            s,rr=copy.deepcopy(summary),copy.deepcopy(rows)
            if failure=='same_tree_twice': next(r['frame'] for r in rr if r.get('frame',{}).get('id')=='h4')['args']['entity']='tree'
            if failure=='missing_log_chain':
                # Two complete tree declarations cannot replace two completed log chains.
                next(r['frame'] for r in rr if r.get('frame',{}).get('id')=='h3')['args']['entity']='tree2'
            if failure=='wrong_deposit': next(r['frame'] for r in rr if r.get('frame',{}).get('type')=='container')['moved']['Wood']=20
            if failure=='under_home': s['finalPos']=[0,25,-10]
            with self.subTest(failure=failure),self.assertRaises(RuntimeError): self.verify(s,rr)

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
