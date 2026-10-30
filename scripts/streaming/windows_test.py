import hashlib
import io
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch
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
