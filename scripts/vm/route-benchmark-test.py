#!/usr/bin/env python3
"""Receipt/arrival and ownership checks without a VM or live movement."""
import copy
import importlib.util
import io
import json
from pathlib import Path
import unittest


def load(name, file):
    spec = importlib.util.spec_from_file_location(name, Path(__file__).with_name(file))
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


benchmark = load('route_benchmark', 'route-benchmark.py')
evidence = load('route_evidence', 'route-evidence.py')
HOME = [0, 37, 0]
MARKER = [10, 37, 0]


class Native:
    def __init__(self):
        self.state = dict(connected=True, phase='in_world', world='one', connection=2, character='Tas',
                          pos=HOME, hp=65, hpMax=65, injurySeq=0, flags=['grounded', 'in_shelter'],
                          location=dict(inWater=False), home=dict(world='one', pos=HOME),
                          food=dict(active=[dict(remainingS=600)]))
        self.calls, self.saved = [], {}
        self.tasks, self.seq = 0, 10
        self.receipt, self.target = None, None
        self.change, self.before_task, self.bad_seq, self.foreign_receipt, self.missing_receipt = None, None, False, False, False

    def call(self, *args):
        self.calls.append(args)
        if args == ('panel', 'GET', '/api/status'):
            if self.target:
                self.state['pos'] = self.target
                if self.change:
                    self.state.update(self.change)
            return copy.deepcopy(self.state)
        if args == ('panel', 'GET', '/api/map'):
            return dict(world='one', connection=2, waypoints=[dict(name='east', pos=MARKER)],
                        perception=dict(seq=(self.seq if self.bad_seq else self.seq+1), characters=[]))
        if args[:2] == ('task', 'move_to'):
            self.tasks += 1
            self.target = json.loads(args[2])['to']
            self.seq += 10
            self.receipt = dict(type='task', re=f't-{self.tasks}', seq=self.seq, status='succeeded')
            if self.before_task:
                self.before_task(self)
            return dict(id=f't-{self.tasks}')
        if args[:2] == ('panel', 'GET') and args[2].startswith('/api/task?'):
            return [] if self.missing_receipt else [dict(conn=99 if self.foreign_receipt else 2, frame=self.receipt)]
        if args == ('panel', 'POST', '/api/act/logout', '{}'):
            return {}
        raise AssertionError(args)

    def save(self, name, value, append=False):
        self.saved[name] = value

    def trial(self):
        self.now = 0
        def pause(_):
            self.now += 1
        return benchmark.Benchmark(self.call, self.save, pause, lambda: self.now)


class Routes(unittest.TestCase):
    def test_two_trips_require_four_scoped_native_receipts_and_final_floor(self):
        native = Native()
        trial = native.trial()
        trial.run(['east'], 2, 30)
        trial.finish(True)
        self.assertEqual(trial.result['outcome'], 'passed')
        self.assertEqual(trial.result['completedTrips'], 2)
        self.assertEqual(len(trial.result['receipts']), 4)
        self.assertTrue(trial.result['keptOnline'])
        self.assertEqual(native.tasks, 4)
        self.assertFalse(any(c[:2] == ('recovery-return', 'arm') for c in native.calls))

    def test_success_text_needs_new_perception_and_measured_arrival(self):
        for mode in ('stale', 'wrong_floor', 'foreign_receipt', 'missing_receipt'):
            with self.subTest(mode=mode):
                native = Native()
                if mode == 'stale': native.bad_seq = True
                if mode == 'wrong_floor': native.change = dict(pos=[10, 34, 0])
                if mode == 'foreign_receipt': native.foreign_receipt = True
                if mode == 'missing_receipt': native.missing_receipt = True
                # Override the fake's native arrival after task dispatch.
                if mode == 'wrong_floor':
                    native.before_task = lambda n: setattr(n, 'target', [10, 34, 0])
                trial = native.trial()
                with self.assertRaises(RuntimeError): trial.run(['east'], 2, 30)
                self.assertEqual(trial.result['completedTrips'], 0)
                self.assertEqual(native.tasks, 1)

    def test_owner_command_cancellation_and_reconnect_revoke_cleanup(self):
        for change in (dict(task=dict(id='owner-order')), dict(connection=3), dict(world='two'),
                       dict(home=dict(world='one', pos=[1, 37, 0]))):
            with self.subTest(change=change):
                native = Native()
                native.change = change
                trial = native.trial()
                with self.assertRaises(RuntimeError): trial.run(['east'], 1, 30)
                trial.finish(False)
                self.assertTrue(trial.yielded)
                self.assertFalse(any('/api/act/logout' in c for c in native.calls))
        native = Native()
        native.before_task = lambda n: n.receipt.update(status='cancelled')
        trial = native.trial()
        with self.assertRaises(RuntimeError): trial.run(['east'], 1, 30)
        trial.finish(False)
        self.assertTrue(trial.yielded)

    def test_start_requires_actual_floor_and_no_foreign_body_task(self):
        for change in (dict(pos=[0, 35.5, 0]), dict(flags=['grounded']), dict(task=dict(id='owner')),
                       dict(connected=False), dict(location=dict(inWater=True))):
            with self.subTest(change=change):
                native = Native()
                native.state.update(change)
                trial = native.trial()
                with self.assertRaises(RuntimeError): trial.run(['east'], 1, 30)
                trial.finish(False)
                self.assertFalse(trial.owned)
                self.assertEqual(native.tasks, 0)

    def test_injury_and_failed_receipt_stop_the_batch(self):
        for mode in ('injury', 'failed'):
            native = Native()
            if mode == 'injury': native.change = dict(injurySeq=1, hp=64)
            else: native.before_task = lambda n: n.receipt.update(status='failed', reason='unreachable')
            trial = native.trial()
            with self.assertRaises(RuntimeError): trial.run(['east'], 3, 30)
            self.assertFalse(trial.yielded)
            self.assertEqual(native.tasks, 1)
            self.assertEqual(trial.result['completedTrips'], 0)
            trial.finish(False)
            self.assertIn(('panel', 'POST', '/api/act/logout', '{}'), native.calls)

    def test_invalid_batch_does_not_contact_vm(self):
        for names, cycles in ((['east'], 11), (['east', 'east'], 1), ([], 1)):
            native = Native()
            with self.assertRaises(RuntimeError): native.trial().run(names, cycles, 30)
            self.assertEqual(native.calls, [])

    def test_a_return_on_the_wrong_floor_does_not_count_a_trip(self):
        native = Native()
        def wrong_return(n):
            if n.tasks == 2: n.target = [0, 35.5, 0]
        native.before_task = wrong_return
        trial = native.trial()
        with self.assertRaisesRegex(RuntimeError, 'target floor'): trial.run(['east'], 2, 30)
        self.assertEqual(trial.result['completedTrips'], 0)
        self.assertEqual(len(trial.result['receipts']), 1)
        self.assertEqual(native.tasks, 2)

    def test_evidence_retains_bounded_support_and_interrupted_search(self):
        rows = []
        for i in range(10):
            rows.append(json.dumps(dict(frame=dict(type='trace', source='walker', msg='navmesh candidate rejected',
                data=dict(supportQuery=dict(reason='slope', contacts=[dict(normal=[0, .62, 0])]))))))
        rows.append('{incomplete')
        rows.append(json.dumps(dict(frame=dict(type='trace', source='walker', msg='checking bounded dry bank departure',
            data=dict(planning=dict(local=dict(expanded=100, failures=dict(counts=dict(endpoint_support=50)))))))) )
        report = evidence.project(io.StringIO('\n'.join(rows)), limit=2)
        self.assertEqual(report['events']['navmesh candidate rejected'], 10)
        self.assertEqual(len(report['recent']), 2)
        self.assertEqual(report['recent'][0]['supportQuery']['reason'], 'slope')
        self.assertEqual(report['recent'][1]['planning']['local']['expanded'], 100)

    def test_evidence_does_not_expand_a_large_bank_sweep_into_context(self):
        sample = dict(supportQuery=dict(reason='slope', contacts=[dict(normal=[0, .62, 0])]*6))
        data = dict(planning=dict(local=dict(failures=dict(counts=dict(endpoint_support=100), examples=[sample]*16))))
        report = evidence.project([json.dumps(dict(frame=dict(type='trace', source='walker', msg='bank', data=data)))])
        sweep = report['recent'][0]['planning']['local']['failures']
        self.assertEqual(len(sweep['examples']), 3)
        self.assertEqual(len(sweep['examples'][0]['supportQuery']['contacts']), 3)
        self.assertEqual(sweep['counts']['endpoint_support'], 100)


if __name__ == '__main__':
    unittest.main()
