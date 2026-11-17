#!/usr/bin/env python3
"""Offline checks of doctor.py's judgements; no VM, SSH or git is used."""
import copy
import importlib.util
from pathlib import Path
import unittest

spec = importlib.util.spec_from_file_location('doctor', Path(__file__).with_name('doctor.py'))
doctor = importlib.util.module_from_spec(spec)
spec.loader.exec_module(doctor)

LOCAL = {'target': 'bot@vm', 'files': {'vm.local': True, 'plugin/game.local.props': True},
         'build': {'in_head': True, 'in_main': True, 'ahead': 0}, 'lease': 'the VM lease is free'}
VM = {
    'boot': '2026-11-04T09:15:26Z', 'desktop': True,
    'tasks': {'RenkiStart': 'Ready', 'RenkiStop': 'Ready', 'RenkiController': 'Ready', 'RenkiOBS': 'Running',
              'RenkiRelay': 'Running', 'RenkiCapture': 'Running', 'RenkiCaptureWatchdog': 'Running'},
    'files': {'bot.local.psd1': True, 'controller.local.json': True, 'secrets.local.json': True},
    'game': {'exe': True, 'bepinex': True, 'plugin': True},
    'controller': {'local': True, 'exe': True, 'running': True, 'answering': True,
                   'bot': {'connected': False, 'phase': ''}},
    'running': {'valheim': False, 'steam': True},
    'tailscale': {'installed': True, 'service': 'Running', 'backend': 'Running'},
    'panel': {'configured': True, 'kind': 'tailscale', 'addressPresent': True, 'http': 200, 'error': ''},
    'deployment': {'version': 'f254e95', 'source': 'f' * 40, 'branch': 't3/x'},
    'diskFreeGB': {'C': 106, 'G': 90},
}


def run(local=None, **changes):
    """Evaluate the healthy VM with dotted-path changes applied."""
    vm = copy.deepcopy(VM)
    for path, value in changes.items():
        node = vm
        *parents, leaf = path.split('__')
        for key in parents:
            node = node[key]
        node[leaf] = value
    checks = doctor.evaluate(copy.deepcopy(local or LOCAL), vm)
    return {c['name']: c for c in checks}, checks


class DoctorTest(unittest.TestCase):
    def test_healthy_offline_vm_passes(self):
        by, checks = run()
        self.assertFalse(doctor.failed(checks))
        self.assertEqual({c['level'] for c in checks}, {doctor.OK, doctor.INFO})
        self.assertEqual(by['bot']['fix'], 'just vm up')
        self.assertIn('all clear', doctor.render(checks))

    def test_missing_target_points_at_setup_and_stops(self):
        checks = doctor.evaluate({'target': '', 'files': {}}, None)
        self.assertEqual([(c['level'], c['fix']) for c in checks], [(doctor.FAIL, 'just setup-worktree')])

    def test_missing_plugin_props_warns_but_still_examines_vm(self):
        local = dict(LOCAL, files={'vm.local': True, 'plugin/game.local.props': False})
        by, checks = run(local)
        self.assertEqual(by['worktree']['level'], doctor.WARN)
        self.assertIn('ssh', by)
        self.assertFalse(doctor.failed(checks))

    def test_ssh_failure_stops_after_reporting_it(self):
        checks = doctor.evaluate(copy.deepcopy(LOCAL), {'error': 'ssh exit 255: timed out'})
        self.assertEqual([c['name'] for c in checks], ['worktree', 'ssh'])
        self.assertTrue(doctor.failed(checks))

    def test_script_failure_is_not_blamed_on_ssh(self):
        checks = doctor.evaluate(copy.deepcopy(LOCAL), {'script_error': 'doctor.ps1 ran on the VM but returned no facts (exit 1)'})
        by = {c['name']: c for c in checks}
        self.assertEqual((by['ssh']['level'], by['doctor']['level']), (doctor.OK, doctor.FAIL))
        self.assertNotIn('sshd', by['doctor']['fix'])

    def test_logged_out_tailscale_is_the_one_panel_finding(self):
        # The state of 2026-11-17: the controller ran, bound to an address
        # that had gone, and every panel request timed out.
        by, checks = run(tailscale__backend='NeedsLogin', panel__addressPresent=False, panel__http=0)
        self.assertEqual(by['tailscale']['level'], doctor.FAIL)
        self.assertIn('sign in', by['tailscale']['fix'])
        self.assertNotIn('panel', by)
        self.assertTrue(doctor.failed(checks))

    def test_loopback_panel_ignores_tailscale(self):
        by, checks = run(panel__kind='loopback', tailscale__backend='NeedsLogin')
        self.assertNotIn('tailscale', by)
        self.assertEqual(by['panel']['level'], doctor.OK)

    def test_unanswering_panel_names_the_restart(self):
        by, _ = run(panel__http=0, panel__error='Timeout')
        self.assertIn('[$addr]', Path(__file__).with_name('doctor.ps1').read_text())
        self.assertEqual(by['panel']['level'], doctor.FAIL)
        self.assertIn('Timeout', by['panel']['detail'])
        self.assertIn('panel-start', by['panel']['fix'])

    def test_unauthorised_panel_reply_still_counts_as_answering(self):
        by, _ = run(panel__http=401)
        self.assertEqual(by['panel']['level'], doctor.OK)

    def test_stopped_controller_warns_without_a_panel_failure(self):
        by, checks = run(controller__running=False, controller__answering=False, panel__http=0)
        self.assertEqual(by['controller']['fix'], 'just vm panel-start')
        self.assertNotIn('panel', by)
        self.assertFalse(doctor.failed(checks))

    def test_no_desktop_tasks_or_plugin_each_fail_with_their_fix(self):
        by, _ = run(desktop=False, tasks={}, game__plugin=False)
        self.assertEqual([by[n]['level'] for n in ('desktop', 'tasks', 'game')], [doctor.FAIL] * 3)
        self.assertIn('release-rdp', by['desktop']['fix'])
        self.assertIn('setup-remote', by['tasks']['fix'])
        self.assertEqual(by['game']['fix'], 'just vm install')

    def test_build_outside_this_branch_warns_but_blocks_up(self):
        # `vm up` copies this worktree's start scripts beside the installed build.
        local = dict(LOCAL, build={'in_head': False, 'in_main': False, 'ahead': 0})
        by, checks = run(local)
        self.assertEqual(by['build']['level'], doctor.WARN)
        self.assertIn('refused', by['build']['detail'])
        self.assertEqual(by['main']['level'], doctor.WARN)
        self.assertFalse(doctor.failed(checks))
        starting = doctor.evaluate(copy.deepcopy(local), copy.deepcopy(VM), starting=True)
        self.assertTrue(doctor.failed(starting))
        self.assertFalse(doctor.failed(doctor.evaluate(copy.deepcopy(LOCAL), copy.deepcopy(VM), starting=True)))

    def test_portless_panel_address_points_at_setup_not_a_restart(self):
        by, _ = run(panel__http=0, panel__error='no port in panel.listen')
        self.assertEqual((by['panel']['level'], by['panel']['fix']), (doctor.FAIL, 'just vm panel-setup'))

    def test_both_local_files_missing_still_warns_about_the_props(self):
        local = dict(LOCAL, files={'vm.local': False, 'plugin/game.local.props': False})
        by, _ = run(local)
        self.assertEqual(by['worktree']['level'], doctor.WARN)

    def test_ssh_failure_text_never_repeats_ssh_output(self):
        self.assertEqual(doctor.ssh_reason('ssh: connect to host 100.101.102.103 port 22: Connection timed out'),
                         'connection timed out')
        self.assertEqual(doctor.ssh_reason('ssh: Could not resolve hostname bot-vm: unknown'),
                         'host name does not resolve')
        self.assertNotIn('bot-vm', doctor.ssh_reason('bot-vm said something odd'))

    def test_unknown_commit_asks_for_a_fetch(self):
        local = dict(LOCAL, build={'in_head': None, 'in_main': None, 'ahead': 0})
        by, checks = run(local)
        self.assertEqual((by['build']['level'], by['build']['fix']), (doctor.WARN, 'git fetch origin'))
        self.assertNotIn('main', by)

    def test_branch_ahead_of_the_installed_build_is_information(self):
        local = dict(LOCAL, build={'in_head': True, 'in_main': True, 'ahead': 3})
        by, _ = run(local)
        self.assertEqual(by['build']['level'], doctor.INFO)
        self.assertIn('3 commit', by['build']['detail'])

    def test_no_receipt_and_low_disk_warn(self):
        by, checks = run(deployment=None, diskFreeGB={'C': 4, 'G': 90})
        self.assertEqual(by['build']['level'], doctor.WARN)
        self.assertIn('C: 4 GB', by['disk']['detail'])
        self.assertFalse(doctor.failed(checks))

    def test_video_partly_running_warns_and_fully_off_informs(self):
        tasks = dict(VM['tasks'], RenkiRelay='Ready')
        by, _ = run(tasks=tasks)
        self.assertEqual(by['video']['level'], doctor.WARN)
        self.assertIn('RenkiRelay', by['video']['detail'])
        off = {name: ('Ready' if name not in ('RenkiStart', 'RenkiStop') else state)
               for name, state in VM['tasks'].items()}
        by, _ = run(tasks=off)
        self.assertEqual(by['video']['level'], doctor.INFO)

    def test_running_game_reports_its_phase(self):
        by, _ = run(running__valheim=True, controller__bot={'connected': True, 'phase': 'in_world'})
        self.assertIn('in_world', by['bot']['detail'])

    def test_script_uses_the_token_only_to_authorise_and_never_reads_secrets(self):
        script = Path(__file__).with_name('doctor.ps1').read_text()
        uses = [line.strip() for line in script.splitlines() if 'Token' in line]
        self.assertEqual(uses, ['if ($settings.Token) { $headers.Authorization = "Bearer $($settings.Token)" }'])
        self.assertNotIn('Get-Content (Join-Path $RenkiDir \'secrets', script)
        self.assertNotRegex(script, r'(?m)^\$(facts|panel)\.\w+ = .*\$(listen|addr|probe)\s*$')

if __name__ == '__main__':
    unittest.main()
