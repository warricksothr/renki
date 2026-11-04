import hashlib
import io
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch
import subprocess
import windows


class PinnedDownloads(unittest.TestCase):
    def test_verified_cache_never_contacts_network(self):
        with tempfile.TemporaryDirectory() as directory:
            out = Path(directory)
            (out / 'asset.zip').write_bytes(b'verified release')
            digest = hashlib.sha256(b'verified release').hexdigest()
            with patch.object(windows, 'OUT', out), patch.object(windows, 'ASSETS', [('asset.zip', 'https://example.invalid', digest)]), patch.object(windows.urllib.request, 'urlopen') as request:
                windows.downloads()
                request.assert_not_called()

    def test_mismatched_release_is_removed_and_never_deployed(self):
        with tempfile.TemporaryDirectory() as directory:
            out = Path(directory)
            with patch.object(windows, 'OUT', out), patch.object(windows, 'ASSETS', [('asset.zip', 'https://example.invalid', '0' * 64)]), patch.object(windows.urllib.request, 'urlopen', return_value=io.BytesIO(b'incorrect release')):
                with self.assertRaises(SystemExit):
                    windows.downloads()
                self.assertFalse((out / 'asset.zip').exists())


class ExistingStackStartup(unittest.TestCase):
    def test_lost_lease_prevents_remote_startup_or_upload(self):
        with patch.dict(windows.os.environ, {'RENKI_VM': 'test@invalid', 'RENKI_VM_DIR': 'C:/renki'}), patch.object(windows.subprocess, 'run', side_effect=subprocess.CalledProcessError(1, 'lease')) as run:
            with self.assertRaises(subprocess.CalledProcessError):
                windows.remote('online')
            self.assertEqual(run.call_count, 1)
            self.assertEqual(run.call_args.args[0][-1], 'guard')

    def test_existing_stack_startup_stages_only_scripts_not_tools_or_credentials(self):
        with patch.dict(windows.os.environ, {'RENKI_VM': 'test@invalid', 'RENKI_VM_DIR': 'C:/renki'}), patch.object(windows.subprocess, 'run') as run, patch.object(windows, 'downloads') as downloads:
            windows.remote('online')
            commands = [call.args[0] for call in run.call_args_list]
            self.assertEqual(commands[0][-1], 'guard')
            upload = next(command for command in commands if command[0] == 'scp')
            self.assertTrue(any(path.endswith('/online.ps1') for path in upload))
            self.assertFalse(any('adapter.json' in path or path.endswith('.exe') for path in upload))
            self.assertTrue(commands[-1][-1].endswith('-Action online'))
            downloads.assert_not_called()
