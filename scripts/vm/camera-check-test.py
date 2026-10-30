import datetime
import importlib.util
from pathlib import Path
import unittest

spec = importlib.util.spec_from_file_location('camera_check', Path(__file__).with_name('camera-check.py'))
camera = importlib.util.module_from_spec(spec)
spec.loader.exec_module(camera)


class CameraCadence(unittest.TestCase):
    def test_normalizes_windows_powershell_api_array_wrapping(self):
        entries = [dict(key='owners', value=['Nimah']), dict(key='screenshot.intervalS', value=5)]
        self.assertEqual(camera.camera_settings(entries), entries[1:])
        self.assertEqual(camera.camera_settings(dict(value=entries, Count=2)), entries[1:])

    def rows(self, fast=(11, 12, 13, 14, 15, 16)):
        return [dict(conn=7, frame=dict(type='screenshot', worldUid='world', capturedAt=datetime.datetime.fromtimestamp(at, datetime.timezone.utc).isoformat(), captureMs=12))
                for at in (0, 5, 10, *fast, 26, 31, 36)]

    def test_requires_actual_scoped_capture_times_in_all_three_stages(self):
        lease = dict(connection=7, worldUid='world', expiresAt='1970-01-01T00:00:26+00:00')
        result = camera.cadence(self.rows(), lease)
        self.assertEqual([result[key]['medianS'] for key in ('before', 'temporary', 'restored')], [5, 1, 5])
        for rows, reason in ((self.rows(fast=(11, 16, 21)), 'too few'),
                             (self.rows(fast=(11, 13, 15, 17)), 'cadence'),
                             (self.rows()[:-2], 'too few')):
            with self.subTest(reason=reason), self.assertRaisesRegex(RuntimeError, reason):
                camera.cadence(rows, lease)
        for key, value, reason in (('worldUid', 'other', 'scope'), ('jpeg', 'bytes', 'leaked')):
            rows = self.rows()
            rows[0]['frame'][key] = value
            with self.subTest(key=key), self.assertRaisesRegex(RuntimeError, reason):
                camera.cadence(rows, lease)


if __name__ == '__main__':
    unittest.main()
