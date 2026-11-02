#!/usr/bin/env python3
"""Bed receipt, native state and command ownership regressions without a VM."""
import copy
import importlib.util
import json
from pathlib import Path
import unittest

spec = importlib.util.spec_from_file_location('sleep_check', Path(__file__).with_name('sleep-check.py'))
sleep = importlib.util.module_from_spec(spec)
spec.loader.exec_module(sleep)


class Native:
    def __init__(self):
        self.now = 0
        self.calls = []
        self.stage = None
        self.refusal = None
        self.attached = True
        self.cancelled = True
        self.receipt_connection = 1
        self.receipt_seq = 20
        self.run = 'one'
        self.age = 0
        self.perception_seq = 31
        self.state = dict(connected=True, phase='in_world', world='world', worldUid='uid', connection=1,
                          character='Tas', pos=[0, 36, 0], hp=25, hpMax=25, flags=['grounded'],
                          time=dict(night=True), owners=['Nimah'])
        self.bed = dict(id='1:bed', kind='bed', mine=True, owned=True, pos=[0, 36, 1])

    def pause(self, seconds):
        self.now += seconds

    def call(self, *args):
        self.calls.append(args)
        if args == ('observe',):
            state = copy.deepcopy(self.state)
            if self.stage == 'escort':
                state.update(task=dict(id='escort', task='escort', status='running'), escort=dict(id='escort'))
            if self.stage == 'sleep' and self.attached and not self.refusal:
                state['flags'] += ['in_bed']
            body = dict(owner='idle')
            if state.get('task'):
                body.update(taskId=state['task']['id'], task=state['task']['task'])
            return dict(status=state, inspection=dict(contract='inspection-v1', run=self.run,
                         world=state['world'], worldUid=state['worldUid'], connection=state['connection'],
                         freshness=dict(status='fresh', ageS=self.age, seq=30), body=body))
        if args == ('panel', 'GET', '/api/map'):
            return dict(world='world', worldUid='uid', connection=1,
                        perception=dict(seq=self.perception_seq, structures=[self.bed]))
        if args[0] == 'task':
            self.stage = 'sleep' if args[1] == 'interact' else 'escort'
            return dict(id=self.stage)
        if args[:2] == ('panel', 'GET') and args[2].startswith('/api/task?'):
            key = 'escort' if 'id=escort' in args[2] else 'sleep'
            status = ('cancelled' if self.cancelled else 'failed') if key == 'escort' else ('failed' if self.refusal else 'succeeded')
            if key == 'escort' and self.stage == 'escort':
                return dict(value=[])
            return dict(value=[dict(conn=self.receipt_connection, frame=dict(type='task', re=key,
                        status=status, seq=self.receipt_seq, reason='refused' if self.refusal else '', detail=self.refusal or ''))])
        raise AssertionError(args)

    def trial(self):
        return sleep.Trial(self.call, lambda *a, **kw: None, self.pause, lambda: self.now)


class Tests(unittest.TestCase):
    def test_actual_bed_entry_can_interrupt_exact_escort(self):
        native = Native()
        result = native.trial().run_trial('Nimah')
        self.assertEqual(result['outcome'], 'succeeded')
        self.assertTrue(result['inBed'])
        self.assertFalse(result['sleeping'])  # Attachment does not promise a shared time skip.
        tasks = [c for c in native.calls if c[0] == 'task']
        self.assertEqual([c[1] for c in tasks], ['escort', 'interact'])
        self.assertEqual(json.loads(tasks[-1][2]), dict(entity='1:bed', sleep=True))

    def test_admission_refuses_without_any_mutation(self):
        for change in (dict(time=dict(night=False)), dict(time=None), dict(task=dict(id='human', task='follow')),
                       dict(connection=0), dict(hp=0), dict(phase='menu')):
            with self.subTest(change=change):
                native = Native()
                native.state.update(change)
                with self.assertRaises(RuntimeError):
                    native.trial().run_trial()
                self.assertFalse(any(c[0] == 'task' for c in native.calls))

    def test_taken_unknown_and_far_beds_are_not_used(self):
        for change in (dict(mine=False), dict(owned=False), dict(owned=None), dict(pos=[30, 36, 0])):
            native = Native()
            native.bed.update(change)
            with self.assertRaises(RuntimeError):
                native.trial().run_trial()
            self.assertFalse(any(c[0] == 'task' for c in native.calls))

    def test_old_perception_cannot_authorize_a_bed_request(self):
        native = Native()
        native.perception_seq = 29
        with self.assertRaisesRegex(RuntimeError, 'new scoped bed perception'):
            native.trial().run_trial()
        self.assertFalse(any(c[0] == 'task' for c in native.calls))

    def test_newer_order_or_run_yields_without_cleanup(self):
        for change in ('order', 'run'):
            native = Native()
            call = native.call
            def changed(*args):
                if native.stage == 'sleep':
                    if change == 'order':
                        native.state['task'] = dict(id='human', task='follow')
                    else:
                        native.run = 'two'
                return call(*args)
            trial = sleep.Trial(changed, lambda *a, **kw: None, native.pause, lambda: native.now)
            with self.assertRaisesRegex(RuntimeError, 'yielded'):
                trial.run_trial()
            self.assertEqual([c[1] for c in native.calls if c[0] == 'task'], ['interact'])

    def test_native_refusal_is_recorded_without_claiming_success(self):
        native = Native()
        native.refusal = 'You are wet'
        result = native.trial().run_trial()
        self.assertEqual(result['outcome'], 'refused')
        self.assertEqual(result['receipt']['detail'], 'You are wet')
        self.assertNotIn('inBed', result)

    def test_success_needs_newer_state_attachment_and_matching_connection(self):
        for name, value in (('attached', False), ('receipt_seq', 30), ('receipt_connection', 2)):
            native = Native()
            setattr(native, name, value)
            with self.assertRaisesRegex(RuntimeError, 'not both verified'):
                native.trial().run_trial(timeout=15)

    def test_escort_needs_exact_cancellation(self):
        native = Native()
        native.cancelled = False
        with self.assertRaisesRegex(RuntimeError, 'cancellation receipt'):
            native.trial().run_trial('Nimah')

    def test_unknown_commander_and_stale_observation_do_not_dispatch(self):
        for leader, age in (('Visitor', 0), ('Nimah', float('nan')), ('Nimah', 4)):
            native = Native()
            native.age = age
            with self.assertRaises(RuntimeError):
                native.trial().run_trial(leader)
            self.assertFalse(any(c[0] == 'task' for c in native.calls))


if __name__ == '__main__':
    unittest.main()
