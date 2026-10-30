#!/usr/bin/env python3
"""Offline contract tests; all OBS/account values are synthetic."""
import contextlib
import copy
import io
import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

import broadcast

ROOT = Path(__file__).resolve().parents[1]
MARKER = "SYNTHETIC_PRIVATE_VALUE"


class BroadcastTests(unittest.TestCase):
    def setUp(self):
        self.profile = broadcast.load(ROOT / "broadcast.example.json")
        self.response = broadcast.load(ROOT / "scripts/fixtures/broadcast/obs-response.json")

    def cli(self, content, response=None):
        with tempfile.TemporaryDirectory() as folder:
            path = Path(folder) / "profile.json"
            path.write_text(content)
            args = ["--profile", str(path)]
            if response is not None:
                path = Path(folder) / "response.json"
                path.write_text(response)
                args += ["--response", str(path)]
            out, err = io.StringIO(), io.StringIO()
            with contextlib.redirect_stdout(out), contextlib.redirect_stderr(err):
                code = broadcast.main(args)
            self.assertNotIn(MARKER, out.getvalue() + err.getvalue())
            return code, out.getvalue(), err.getvalue()

    def test_plan_is_read_only_and_inert(self):
        with patch("socket.socket", side_effect=AssertionError("network")), \
                patch("subprocess.Popen", side_effect=AssertionError("execution")), \
                patch.dict("os.environ", {"TWITCH_STREAM_KEY": MARKER, "RENKI_PANEL_TOKEN": MARKER}):
            code, text, _ = self.cli(json.dumps(self.profile))
        self.assertEqual(code, 0)
        value = json.loads(text)
        self.assertEqual(value["mode"], "offline-plan")
        batch = value["diagnosticBatch"]
        self.assertEqual(batch["op"], 8)
        self.assertEqual(batch["d"]["executionType"], 0)
        self.assertEqual([r["requestType"] for r in batch["d"]["requests"]],
                         ["GetVersion", "GetVideoSettings", "GetStreamStatus", "GetStats", "GetSourceActive"])

    def test_supported_target_variants(self):
        for size in ((1280, 720), (1920, 1080)):
            for fps in (30, 60):
                self.profile["video"].update(width=size[0], height=size[1], fps=fps)
                self.profile["capture"].update(method="window", encoder="software-h264")
                self.profile["audio"]["source"] = "game-only"
                self.profile["overlay"]["enabled"] = True
                broadcast.validate(self.profile)

    def test_reject_unknown_fields_and_secrets_without_echo(self):
        for path in ((), ("video",), ("audio",), ("capture",), ("overlay",)):
            profile = copy.deepcopy(self.profile)
            target = profile[path[0]] if path else profile
            target[MARKER] = MARKER
            code, out, err = self.cli(json.dumps(profile))
            self.assertEqual(code, 2)
            self.assertEqual(out, "")
            self.assertIn("schema", err)

    def test_reject_invalid_targets_and_types(self):
        cases = [
            ("video", "fps", True), ("video", "fps", 29.97),
            ("video", "fps", 1), ("video", "bitrateKbps", 2499),
            ("video", "bitrateKbps", 6001), ("video", "keyframeSeconds", 1),
            ("video", "width", 1920), ("video", "height", 721),
            ("capture", "method", "display"), ("capture", "encoder", MARKER),
            ("audio", "source", "desktop"), ("audio", "bitrateKbps", 161),
            ("overlay", "enabled", "true"),
        ]
        for group, key, value in cases:
            with self.subTest(group=group, key=key, value=value):
                profile = copy.deepcopy(self.profile)
                profile[group][key] = value
                self.assertEqual(self.cli(json.dumps(profile))[0], 2)
        for value in (None, [], {}, {"version": True}, MARKER):
            self.assertEqual(self.cli(json.dumps(value))[0], 2)

    def test_reject_malformed_duplicate_nonfinite_and_large_json(self):
        for content in ('{"' + MARKER, '{"version":1,"version":2}',
                        '{"video":NaN}', '{"video":Infinity}', ' ' * (256 * 1024 + 1)):
            self.assertEqual(self.cli(content)[0], 2)

    def test_missing_file_is_sanitized(self):
        out, err = io.StringIO(), io.StringIO()
        with contextlib.redirect_stdout(out), contextlib.redirect_stderr(err):
            code = broadcast.main(["--profile", "/nonexistent/" + MARKER])
        self.assertEqual(code, 2)
        self.assertNotIn(MARKER, err.getvalue())

    def test_invalid_cli_arguments_are_not_echoed(self):
        for args in ([MARKER], ["--" + MARKER, MARKER], ["--profile"]):
            out, err = io.StringIO(), io.StringIO()
            with contextlib.redirect_stdout(out), contextlib.redirect_stderr(err):
                self.assertEqual(broadcast.main(args), 2)
            self.assertEqual(out.getvalue(), "")
            self.assertNotIn(MARKER, err.getvalue())

    def test_fixture_and_output_allowlist(self):
        for result in self.response["d"]["results"]:
            result["responseData"][MARKER] = MARKER
            result["requestStatus"]["comment"] = MARKER
        self.response["d"][MARKER] = MARKER
        code, text, _ = self.cli(json.dumps(self.profile), json.dumps(self.response))
        self.assertEqual(code, 0)
        value = json.loads(text)
        self.assertEqual(value["captureHealth"], "unverified")
        self.assertTrue(value["capture"]["inProgram"])
        self.assertEqual(value["stream"]["state"], "inactive")
        self.assertTrue(value["video"]["matchesTarget"])
        self.assertTrue(value["version"]["diagnosticsSupported"])

    def test_stream_reconnect_is_not_capture_health(self):
        stream = self.response["d"]["results"][2]["responseData"]
        for active, reconnect, state in ((True, False, "active"), (True, True, "reconnecting"),
                                         (False, True, "reconnecting")):
            stream.update(outputActive=active, outputReconnecting=reconnect)
            value = broadcast.summarize(self.profile, self.response)
            self.assertEqual(value["stream"]["state"], state)
            self.assertEqual(value["captureHealth"], "unverified")

    def test_failed_request_keeps_only_fixed_label_and_code(self):
        result = self.response["d"]["results"][4]
        result["requestStatus"] = {"result": False, "code": 600, "comment": MARKER}
        result["responseData"] = MARKER
        code, text, _ = self.cli(json.dumps(self.profile), json.dumps(self.response))
        self.assertEqual(code, 0)
        self.assertEqual(json.loads(text)["capture"], {"ok": False, "code": 600})

    def test_video_mismatch_and_missing_capability(self):
        self.response["d"]["results"][0]["responseData"]["availableRequests"] = []
        self.response["d"]["results"][1]["responseData"]["outputWidth"] = 1920
        value = broadcast.summarize(self.profile, self.response)
        self.assertFalse(value["video"]["matchesTarget"])
        self.assertFalse(value["version"]["diagnosticsSupported"])

    def test_reject_incomplete_duplicate_wrong_identity_or_type(self):
        variants = []
        for key, value in (("requestId", MARKER), ("requestType", MARKER)):
            response = copy.deepcopy(self.response)
            response["d"]["results"][0][key] = value
            variants.append(response)
        response = copy.deepcopy(self.response)
        response["d"]["results"].pop()
        variants.append(response)
        response = copy.deepcopy(self.response)
        response["d"]["results"][0] = response["d"]["results"][1]
        variants.append(response)
        variants += [{"op": 7}, {"op": 9, "d": {"requestId": MARKER}}]
        for response in variants:
            self.assertEqual(self.cli(json.dumps(self.profile), json.dumps(response))[0], 2)

    def test_reject_invalid_response_numbers_and_boolean_coercion(self):
        for key, value in (("outputBytes", -1), ("outputBytes", True),
                           ("outputBytes", MARKER), ("outputActive", 1)):
            response = copy.deepcopy(self.response)
            response["d"]["results"][2]["responseData"][key] = value
            self.assertEqual(self.cli(json.dumps(self.profile), json.dumps(response))[0], 2)


if __name__ == "__main__":
    unittest.main()
