#!/usr/bin/env python3
"""Validate a secret-free broadcast target; plan/inspect OBS v5 diagnostics offline.

No network, subprocess, environment, credentials or OBS mutation support. All
output is constructed from validated numbers, booleans and fixed vocabulary.
"""
import argparse
import json
import math
from pathlib import Path
import sys

ROOT = Path(__file__).resolve().parents[1]
BATCH_ID = "renki-broadcast-diagnostics"
REQUESTS = (
    ("version", "GetVersion", {}),
    ("video", "GetVideoSettings", {}),
    ("stream", "GetStreamStatus", {}),
    ("stats", "GetStats", {}),
    ("capture", "GetSourceActive", {"sourceName": "Renki Game Capture"}),
)


class Invalid(ValueError):
    """A fixed, non-sensitive diagnostic that is safe for task logs."""


class SafeParser(argparse.ArgumentParser):
    def error(self, _message):
        raise Invalid("Unsupported or incomplete arguments; use --help")


def require(condition, message):
    if not condition:
        raise Invalid(message)


def fields(value, names):
    require(type(value) is dict and set(value) == set(names.split()),
            "Object fields do not match the supported schema")


def integer(value, low, high):
    return type(value) is int and low <= value <= high


def validate(profile):
    fields(profile, "version video audio capture overlay")
    require(type(profile["version"]) is int and profile["version"] == 1,
            "Unsupported profile version")
    video = profile["video"]
    fields(video, "width height fps bitrateKbps keyframeSeconds")
    require(integer(video["width"], 1280, 1920)
            and integer(video["height"], 720, 1080)
            and (video["width"], video["height"]) in ((1280, 720), (1920, 1080)),
            "Video size must be 1280x720 or 1920x1080")
    require(integer(video["fps"], 30, 60) and video["fps"] in (30, 60),
            "Video FPS must be 30 or 60")
    require(integer(video["bitrateKbps"], 2500, 6000),
            "Video bitrate must be 2500-6000 Kbps")
    require(type(video["keyframeSeconds"]) is int and video["keyframeSeconds"] == 2,
            "Keyframe interval must be 2 seconds")
    audio = profile["audio"]
    fields(audio, "source bitrateKbps")
    require(audio["source"] in ("off", "game-only"), "Unsupported audio source")
    require(integer(audio["bitrateKbps"], 96, 160), "Audio bitrate must be 96-160 Kbps")
    capture = profile["capture"]
    fields(capture, "method encoder")
    require(capture["method"] in ("game", "window"), "Unsupported capture method")
    require(capture["encoder"] in ("hardware-h264", "software-h264"),
            "Unsupported encoder preference")
    fields(profile["overlay"], "enabled")
    require(type(profile["overlay"]["enabled"]) is bool, "Overlay enabled must be boolean")
    return profile


def plan(profile):
    validate(profile)
    return {
        "mode": "offline-plan",
        "target": profile,
        "encoding": {"codec": "h264", "rateControl": "CBR", "audioCodec": "AAC"},
        "scenes": {"game": "Renki Gameplay", "fallback": "Renki Offline"},
        "diagnosticBatch": {
            "op": 8,
            "d": {
                "requestId": BATCH_ID,
                "haltOnFailure": False,
                "executionType": 0,
                "requests": [
                    {"requestId": "renki-" + name, "requestType": kind, "requestData": data}
                    for name, kind, data in REQUESTS
                ],
            },
        },
    }


def number(data, key):
    value = data.get(key)
    require(type(value) in (int, float) and math.isfinite(value)
            and 0 <= value <= 2**53, "Invalid OBS numeric field")
    return value


def boolean(data, key):
    value = data.get(key)
    require(type(value) is bool, "Invalid OBS boolean field")
    return value


def summarize(profile, response):
    validate(profile)
    require(type(response) is dict and response.get("op") == 9,
            "Expected OBS RequestBatchResponse (op 9)")
    batch = response.get("d")
    require(type(batch) is dict and batch.get("requestId") == BATCH_ID,
            "OBS batch identity mismatch")
    results = batch.get("results")
    require(type(results) is list and len(results) == len(REQUESTS),
            "OBS batch is incomplete")
    expected = {"renki-" + name: (name, kind) for name, kind, _ in REQUESTS}
    output = {"mode": "offline-diagnostics", "captureHealth": "unverified"}
    seen = set()
    for result in results:
        require(type(result) is dict, "Invalid OBS result")
        identity = result.get("requestId")
        require(type(identity) is str and identity in expected and identity not in seen,
                "Unexpected or duplicate OBS result")
        seen.add(identity)
        name, kind = expected[identity]
        require(result.get("requestType") == kind, "OBS request type mismatch")
        status = result.get("requestStatus")
        require(type(status) is dict and type(status.get("result")) is bool
                and integer(status.get("code"), 0, 9999), "Invalid OBS request status")
        if not status["result"]:
            output[name] = {"ok": False, "code": status["code"]}
            continue  # Never log OBS error comments or failed response payloads.
        require(status["code"] == 100, "Inconsistent OBS success status")
        data = result.get("responseData")
        require(type(data) is dict, "Missing OBS response data")
        value = {"ok": True}
        if name == "version":
            available = data.get("availableRequests")
            require(type(available) is list and all(type(v) is str for v in available)
                    and integer(data.get("rpcVersion"), 1, 100), "Invalid OBS capabilities")
            value["diagnosticsSupported"] = all(kind in available for _, kind, _ in REQUESTS)
        elif name == "video":
            for key in ("baseWidth", "baseHeight", "outputWidth", "outputHeight",
                        "fpsNumerator", "fpsDenominator"):
                require(integer(data.get(key), 1, 100000), "Invalid OBS video settings")
                value[key] = data[key]
            target = profile["video"]
            value["matchesTarget"] = (
                data["outputWidth"] == target["width"]
                and data["outputHeight"] == target["height"]
                and data["fpsNumerator"] == target["fps"] * data["fpsDenominator"]
            )
        elif name == "stream":
            active, reconnecting = boolean(data, "outputActive"), boolean(data, "outputReconnecting")
            value["state"] = "reconnecting" if reconnecting else "active" if active else "inactive"
            for key in ("outputBytes", "outputDuration", "outputCongestion",
                        "outputSkippedFrames", "outputTotalFrames"):
                value[key] = number(data, key)
        elif name == "stats":
            for key in ("activeFps", "averageFrameRenderTime", "renderSkippedFrames",
                        "renderTotalFrames", "outputSkippedFrames", "outputTotalFrames"):
                value[key] = number(data, key)
        elif name == "capture":
            value["inProgram"] = boolean(data, "videoActive")
            value["showingInUI"] = boolean(data, "videoShowing")
        output[name] = value
    return output


def reject_constant(_):
    raise Invalid("Non-finite JSON numbers are not supported")


def unique_object(pairs):
    value = {}
    for key, item in pairs:
        require(key not in value, "Duplicate JSON field")
        value[key] = item
    return value


def load(path):
    # Bound accidental full audit/recording reads. Never include a path or JSON
    # value in errors: either could contain a token supplied by mistake.
    with path.open("rb") as stream:
        content = stream.read(256 * 1024 + 1)
    require(len(content) <= 256 * 1024, "JSON input exceeds 256 KiB")
    return json.loads(content, parse_constant=reject_constant, object_pairs_hook=unique_object)


def main(argv=None):
    parser = SafeParser(description=__doc__)
    parser.add_argument("--profile", type=Path, default=ROOT / "broadcast.example.json")
    parser.add_argument("--response", type=Path, help="local synthetic OBS op 9 response JSON")
    try:
        args = parser.parse_args(argv)
        profile = validate(load(args.profile))
        result = summarize(profile, load(args.response)) if args.response else plan(profile)
        print(json.dumps(result, indent=2, allow_nan=False))
    except Invalid as error:
        print("broadcast: " + str(error), file=sys.stderr)
        return 2
    except (OSError, ValueError, TypeError, RecursionError, OverflowError):
        print("broadcast: Unable to read supported JSON input", file=sys.stderr)
        return 2
    return 0


if __name__ == "__main__":
    sys.exit(main())
