#!/usr/bin/env python3
"""Capture a scoped map bundle and self-contained HTML using the panel renderer."""
import argparse
import datetime
import json
from pathlib import Path
import subprocess
import urllib.parse

ROOT = Path(__file__).resolve().parents[2]
parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument('--html', action='store_true', help='emit the self-contained render document for a headless image tool')
args = parser.parse_args()
folder = ROOT / '.checks' / ('map-snapshot-' + datetime.datetime.now(datetime.timezone.utc).strftime('%Y%m%dT%H%M%S%fZ'))
folder.mkdir(parents=True)

def get(name, endpoint):
    result = subprocess.run(['just','vm','panel','GET',endpoint],cwd=ROOT,capture_output=True,text=True,timeout=45)
    (folder/(name+'.log')).write_text(result.stderr)
    result.check_returncode()
    value=json.loads(result.stdout)
    (folder/(name+'.json')).write_text(json.dumps(value,indent=2)+'\n')
    return value

status=get('status','/api/status')
mapped=get('map','/api/map')
if status.get('phase')!='in_world' or status.get('world')!=mapped.get('world') or status.get('connection')!=mapped.get('connection'):
    raise SystemExit('A live map snapshot requires matching current world/connection; use the panel export for a saved/offline atlas')
x,_,z=status['pos']
query=urllib.parse.urlencode(dict(world=mapped['worldKey'],minX=x-160,maxX=x+160,minZ=z-160,maxZ=z+160))
get('atlas','/api/atlas/map?'+query)
after=get('after','/api/status')
if (after.get('world'),after.get('connection'),after.get('phase'))!=(status['world'],status['connection'],'in_world'):
    raise SystemExit('Map scope changed during capture; bundle is incomplete')
output=folder/'map.html'
subprocess.run(['just','controller','preview-map','--status',str(folder/'status.json'),'--map',str(folder/'map.json'),'--atlas',str(folder/'atlas.json'),'--map-only','--output',str(output)],cwd=ROOT,check=True,capture_output=True,text=True)
metadata=dict(kind='map',capturedAt=datetime.datetime.now(datetime.timezone.utc).isoformat(),world=status['world'],connection=status['connection'],worldKey=mapped.get('worldKey'),pos=status['pos'],html=str(output),limitations=['Sequential scoped observations, not an atomic world photograph','Read-only render of the panel default viewport; use the panel export for its current pan/zoom/layers'])
(folder/'metadata.json').write_text(json.dumps(metadata,indent=2)+'\n')
if args.html:
    print(output.read_text())
else:
    print('Map render: '+str(output))
    print('Data and metadata: '+str(folder))
