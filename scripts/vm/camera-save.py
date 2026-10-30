#!/usr/bin/env python3
"""Save one camera image and metadata; never print image bytes into context."""
import base64
import datetime
import json
from pathlib import Path
import sys

value = json.load(sys.stdin)
data = base64.b64decode(value.pop('jpegBase64'), validate=True)
if not 0 < len(data) <= 512 * 1024 or not data.startswith(b'\xff\xd8'):
    raise SystemExit('Camera response is not a bounded JPEG')
root = Path(__file__).resolve().parents[2] / '.checks'
root.mkdir(exist_ok=True)
name = 'camera-' + datetime.datetime.now(datetime.timezone.utc).strftime('%Y%m%dT%H%M%S%fZ')
image = root / (name + '.jpg')
image.write_bytes(data)
image.with_suffix('.json').write_text(json.dumps(value, indent=2) + '\n')
frame = value['frame']
print(f"Camera: {frame['width']}x{frame['height']}, {len(data)} bytes, capture {frame['captureMs']:.1f} ms, age {value['ageS']:.1f} s, task {frame.get('task') or 'idle'}")
print('Image: ' + str(image))
print('Metadata: ' + str(image.with_suffix('.json')))
