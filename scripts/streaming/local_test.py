import unittest
from local import configuration


class PrivateRelayTests(unittest.TestCase):
    def test_loopback_only_no_destinations_or_extra_protocols(self):
        c = configuration('server-side', '/tmp/slate.mp4')
        for key in ['rtspAddress', 'rtmpAddress', 'webrtcAddress', 'webrtcLocalUDPAddress', 'webrtcLocalTCPAddress']:
            self.assertTrue(c[key].startswith('127.0.0.1:'), key)
        for key in ['api', 'metrics', 'pprof', 'playback', 'hls', 'srt', 'moq', 'webrtcIPsFromInterfaces']:
            self.assertFalse(c[key], key)
        self.assertEqual(c['webrtcICEServers2'], [])
        self.assertEqual(list(c['paths']), ['renki'])
        path = c['paths']['renki']
        self.assertEqual(path['forward'], [])
        self.assertFalse(path['record'])
        self.assertTrue(path['alwaysAvailable'])
        self.assertEqual(path['maxReaders'], 4)
        self.assertFalse(path['overridePublisher'])

    def test_anonymous_publish_is_loopback_only_read_requires_panel_credential(self):
        c = configuration('server-side', '/tmp/slate.mp4')
        publish, read = c['authInternalUsers']
        self.assertEqual(publish['ips'], ['127.0.0.1'])
        self.assertEqual(publish['permissions'], [dict(action='publish', path='renki')])
        self.assertEqual(read['user'], 'panel')
        self.assertEqual(read['pass'], 'server-side')
        self.assertEqual(read['permissions'], [dict(action='read', path='renki')])
