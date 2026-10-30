"""Build a read-only preview of the real panel assets, optionally serving it.

Default terrain is a fixture. --status/--map use captured API snapshots.
No credentials, game tasks or external requests; embeds the production bundle.
"""
import argparse
import http.server
import json,pathlib,math

parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument('--suite',action='store_true',help='Deterministic recorded UI suite, local font and read-only API; requires just panel install')
parser.add_argument('--status')
parser.add_argument('--map')
parser.add_argument('--camera',help='Recorded JPEG, replayed with fixture timing; never a live capture')
parser.add_argument('--camera-state',choices=['image','disabled','pending','stale','failed','broken','temporary'],default='image')
parser.add_argument('--offline',action='store_true')
parser.add_argument('--serve', action='store_true')
parser.add_argument('--output')
parser.add_argument('--atlas')
parser.add_argument('--map-only', action='store_true')
parser.add_argument('--port', type=int, default=18780)
args = parser.parse_args()
if bool(args.status) != bool(args.map):
 parser.error('--status and --map must be supplied together')
root=pathlib.Path(__file__).resolve().parent.parent
if args.suite:
 args.status=str(root/'panel/fixtures/status.json')
 args.map=str(root/'panel/fixtures/mapped.json')
 args.atlas=str(root/'panel/fixtures/atlas.json')
 args.camera=str(root/'panel/fixtures/camera.jpg')
status=dict(connected=True,connection=1,character='Tasleroch',world='DizzysHellheim',phase='in_world',hp=25,hpMax=25,biome='Meadows',pos=[-36.72,36.09,8.03],yaw=40,perceptionRadius=40,flags=['grounded','in_shelter'],location=dict(groundHeight=33.2,support='structure',inWater=False,waterDepth=0),owners=['Nimah'],home=dict(pos=[-35.9,37.66,6.53],distance=2))
tiles=[]
for tz in range(-3,4):
 for tx in range(-6,1):
  heights=[];biomes=[]
  for z in range(5):
   for x in range(5):
    wx,wz=tx*16+x*4,tz*16+z*4
    heights.append(round(31-(wx+15)*.12+2*math.sin(wz/20),1));biomes.append('BlackForest' if wz>40 else 'Meadows')
  tiles.append(dict(x=tx,z=tz,heights=heights,biomes=biomes,waterLevel=30))
landmarks=[dict(entity='old-station',kind='station',prefab='piece_workbench',pos=[-61,33,0],at='2026-10-29T18:00:00Z')]
mapped=dict(world=status['world'],connection=1,tiles=tiles,landmarks=landmarks)
mapped['danger'] = [dict(pos=[-65,33,18],radius=12,score=2.5,hits=3,deaths=0,at='2026-10-30T12:00:00Z',prefab='Greyling')]
mapped['waypoints'] = [dict(name='storage',kind='storage',pos=[-34,36,4],radius=6,source='bot',setBy='bot',at='2026-10-29T18:00:00Z',evidence='Observed chest'),dict(name='woodlot',kind='wood',pos=[-53,33,25],radius=12,source='owner',setBy='Nimah',at='2026-10-29T18:01:00Z')]
frames=[dict(type='layout',walls=[dict(a=[-42,36,-2],b=[-34,36,-5],h=2),dict(a=[-34,36,-5],b=[-27,36,13],h=2),dict(a=[-27,36,13],b=[-35,36,16],h=2),dict(a=[-35,36,16],b=[-42,36,-2],h=2)]),dict(type='perception',structures=[dict(id='chest',kind='container',pos=[-34,36,4]),dict(id='door',kind='door',open=True,pos=[-40,36,3]),dict(id='bench',kind='station',pos=[-30,36,9]),dict(id='bed',kind='bed',pos=[-34,36,12])],resources=[dict(id='tree',kind='tree',prefab='Beech1',pos=[-53,33,25])])]
mapped['walls'] = frames[0]['walls']
mapped['perception'] = frames[1]
mapped['containers'] = [dict(entity='chest',pos=[-34,36,4],size=[5,2],at='2026-10-29T18:00:00Z',items=[dict(prefab='CookedDeerMeat',stack=9),dict(prefab='Wood',stack=12)]),dict(entity='empty-chest',pos=[-54,33,4],size=[5,2],at='2026-10-29T17:00:00Z',items=[])]
frames[1]['structures'].extend([dict(id='empty-chest',kind='container',pos=[-54,33,4]),dict(id='unknown-chest',kind='container',pos=[-54,33,14])])
if args.status:
 status=json.loads(pathlib.Path(args.status).read_text())
 mapped=json.loads(pathlib.Path(args.map).read_text())
 frames=[]
# Preview an accumulated saved map independently of the live connection.
import copy,datetime
manifest=mapped.get('manifest') or {}
atlas_key=manifest.get('key') or mapped.get('worldKey') or 'a'*64
position=status.get('pos') or [-36,0,8]
saved_world=dict(key=atlas_key,name=manifest.get('name') or status.get('world') or mapped.get('world') or 'Saved world',scope=manifest.get('scope') or dict(namespace='preview',worldUid='9223372036854775807',epoch='1'),center=manifest.get('center') or [position[0],position[2]])
saved_tiles=copy.deepcopy(mapped.get('tiles',[]))
for tile in saved_tiles:
 tile.setdefault('observedAt',['2026-10-01T12:00:00Z']*25)
 tile.setdefault('sources',['preview-observer']*25)
 tile.setdefault('stale',[tile['x'] < -4]*25)
mapped['worldKey']=atlas_key
saved_map=json.loads(pathlib.Path(args.atlas).read_text()) if args.atlas else dict(worldKey=atlas_key,world=status.get('world',''),connection=status.get('connection',0),tiles=saved_tiles)
# Everything below embeds the built production assets, including the Preact
# workspace and the same renderer used by window.renkiSnapshots.map().
import base64,gzip
camera_jpeg=base64.b64encode(pathlib.Path(args.camera).read_bytes()).decode() if args.camera else ''
frames.append(dict(type='inventory',size=[8,4],items=[dict(slot=[0,0],prefab='AxeBronze',stack=1,durability=84,durabilityMax=100,equipped=True),dict(slot=[1,0],prefab='CookedDeerMeat',stack=9,stackMax=20),dict(slot=[2,0],prefab='Wood',stack=12,stackMax=50)]))
if not args.status:
 status.update(hp=74,hpMax=100,stamina=65,staminaMax=90,food=dict(active=[dict(prefab='CookedDeerMeat',remainingS=740),dict(prefab='Raspberry',remainingS=350)],carried=[]),task=dict(task='move_to',id='t-preview'),combat=dict(mode='off',reason='combat disabled'))
status.setdefault('owners',['Nimah'])
if args.offline:
 status=dict(connected=False,connection=status.get('connection',1),owners=status['owners'],phase='menu')
fixtures=dict(status=status,mapped=mapped,frames=frames,atlas=saved_map,worlds=[saved_world],jpeg=camera_jpeg,cameraState=args.camera_state,settings=[dict(key=key,value=value,config=value) for key,value in [('voice',False),('owners',['Nimah']),('screenshot.enabled',bool(camera_jpeg)),('screenshot.intervalS',5),('screenshot.width',960),('screenshot.quality',70),('presence.sharePosition',False),('players.profiles',{}),('home.afterDeath','home')]],errors=[],requests=[],records=[
 dict(kind='frame',cat='task',task='t-preview',at='2026-10-30T18:50:00Z',frame=dict(type='task',re='t-preview',task='move_to',status='failed',reason='path_blocked'),line='18:50:00 task t-preview move_to failed reason=path_blocked trace={"navmesh":false}'),
 dict(kind='log',cat='chat',at='2026-10-30T18:50:02Z',msg='Tas: The path is blocked. I stopped safely.',line='18:50:02 said to the panel text="The path is blocked. I stopped safely."'),
 dict(kind='frame',cat='trace',task='t-preview',at='2026-10-30T18:50:01Z',frame=dict(type='trace',msg='stuck recovery',source='walker',data=dict(attempt=2)),line='18:50:01 trace task=t-preview source=walker msg="stuck recovery" data={"attempt":2}')])

def embedded(value):
 return json.dumps(value,separators=(',',':'))

# Compress only fixture transport. Decompression changes no map data or renderer.
packed=base64.b64encode(gzip.compress(embedded(fixtures).encode(),mtime=0)).decode()
bootstrap='''<script>
window.renkiPreviewReady=(async()=>{
const packed=Uint8Array.from(atob('PACKED'),c=>c.charCodeAt(0));
window.renkiPreview=await new Response(new Blob([packed]).stream().pipeThrough(new DecompressionStream('gzip'))).json();
const preview=window.renkiPreview;
preview.temporaryExpires=Date.now()+30000;
preview.cameraSequence=1;
preview.setCamera=(state)=>{preview.cameraState=state;preview.cameraSequence++;};

const cameraPath='/api/'+'camera.jpg';
window.addEventListener('error',e=>preview.errors.push(e.message));
window.addEventListener('unhandledrejection',e=>preview.errors.push(String(e.reason)));
const reply=(value,status=200)=>new Response(JSON.stringify(value),{status,headers:{'Content-Type':'application/json'}});
window.fetch=async(path,options={})=>{
 const request={path:String(path),method:options.method || 'GET',headers:options.headers || {},body:options.body || null,status:200};
 preview.requests.push(request);
 const respond=(value,status=200)=>{request.status=status;return reply(value,status);};
 if (options.method && options.method!=='GET') return respond({text:'Read-only preview: actions are refused'},405);
 const url=new URL(path,location.href);
 if(url.origin!==location.origin){const message='External fixture fetch refused: '+url.origin;preview.errors.push(message);throw Error(message);}
 switch(url.pathname){
 case '/api/status': return respond(preview.status);
 case '/api/map': return respond(preview.mapped);
 case '/api/atlas/worlds': return respond(preview.worlds);
 case '/api/atlas/map': return respond(preview.atlas);
 case '/api/task': return respond(preview.records);
 case '/api/settings': return respond(preview.settings);
 case '/api/models': return respond({models:['fixture-model']});
 case '/api/days': return respond([{day:'2026-10-30',bytes:150000}]);
 case '/api/timeline': return respond({events:preview.records,more:!url.searchParams.has('before')});
 case '/api/inventory': return respond({world:preview.status.world,connection:preview.status.connection,weight:42,weightMax:300,inventory:preview.frames.find(f=>f.type==='inventory')});
 case '/api/camera': {
  if(preview.cameraState==='failed')return respond({text:'Fixture camera unavailable'},503);
  if(preview.cameraState==='temporary' && Date.now()>=preview.temporaryExpires)preview.cameraState='image';
  const available=!!preview.jpeg && !['disabled','pending'].includes(preview.cameraState);
  return respond({available,config:{enabled:preview.cameraState!=='disabled' && !!preview.jpeg,intervalS:preview.cameraState==='temporary'?1:5},baseConfig:{enabled:!!preview.jpeg,intervalS:5},override:preview.cameraState==='temporary'?{id:'fixture-override',config:{enabled:true,intervalS:1},expiresAt:new Date(preview.temporaryExpires).toISOString(),connection:preview.status.connection,worldUid:preview.status.worldUid}:null,ageS:preview.cameraState==='stale'?60:2,imageUrl:cameraPath+'?connection='+preview.status.connection+'&seq='+preview.cameraSequence,frame:{...preview.status,seq:preview.cameraSequence,capturedAt:new Date(Date.now()-(preview.cameraState==='stale'?65000:3000)).toISOString(),width:960,height:540,captureMs:4.3,task:'fixture replay'}});
 }
 case cameraPath: return new Response(new Blob([Uint8Array.from(atob(preview.cameraState==='broken'?'aW52YWxpZA==':preview.jpeg),c=>c.charCodeAt(0))],{type:'image/jpeg'}));
 default: {const message='Unexpected fixture request: '+url.pathname;preview.errors.push(message);throw Error(message);}
 }
};
window.EventSource=class{
 constructor(){this.handlers={};preview.stream=this;setTimeout(()=>{this.onopen?.();for(const frame of preview.frames)this.emit('record',{kind:'frame',cat:'perception',frame});for(const record of preview.records)this.emit('record',record);this.emit('status',preview.status);},20);}
 addEventListener(name,fn){this.handlers[name]=fn;}
 emit(name,value){if(name==='status')preview.status=value;this.handlers[name]?.({data:JSON.stringify(value)});}
};
})();
</script>'''.replace('PACKED',packed)
html=(root/'internal/panel/static/index.html').read_text()
# Mirror the production image policy so blob-image regressions are visible.
html=html.replace('<head>','<head><meta http-equiv="Content-Security-Policy" content="default-src \'none\'; script-src \'unsafe-inline\'; style-src \'unsafe-inline\'; img-src \'self\' data: blob:; connect-src \'self\';">')
html=html.replace('<link rel="stylesheet" href="/style.css">','<style>'+(root/'internal/panel/static/style.css').read_text()+'</style>')
html=html.replace('<script src="/terrain.js" defer></script>',bootstrap+'<script>'+(root/'internal/panel/static/terrain.js').read_text()+'</script>')
html=html.replace('<script src="/app.js" defer></script>','')
notice='Recorded game assets · read-only fixture replay (not live)' if args.status or args.camera else 'Fixture terrain · read-only preview (not live)'
html=html.replace('<body>','<body><aside class="preview-notice" style="padding:8px 16px;color:#d2dfc6;background:#35452b;font:12px system-ui">'+notice+'</aside>')
checks='''<script>window.renkiPreviewReady.then(()=>setTimeout(()=>{try{const shot=window.renkiSnapshots.map({maxWidth:640,quality:.8});console.log(JSON.stringify({snapshot:shot.metadata,errors:renkiPreview.errors,canvas:document.getElementById('map').getBoundingClientRect().toJSON()}));const layer=document.querySelector('[data-layer=terrain]');layer.click();if(window.renkiSnapshots.map().metadata.layers.terrain!==false)throw Error('Terrain toggle failed');layer.click();console.log('Shared renderer and layer controls passed');}catch(e){renkiPreview.errors.push(e.message);console.error(e);}},250));</script>'''
if args.suite:
 checks='' # Automation owns interactions; no delayed self-check can toggle its controls.
html=html.replace('</body>','<script>window.renkiPreviewReady.then(()=>{'+(root/'panel/preview-checks.js').read_text()+'\n'+(root/'internal/panel/static/app.js').read_text()+'\n}).catch(error=>{document.getElementById("app").textContent="Preview could not load: "+error.message;console.error(error)});</script>'+checks+'</body>')
if args.map_only:
 html=html.replace('</head>','<style>.topbar,.overview,.workspace-title,.operations,#feed-pane,#camera,.command-bar,footer,.preview-notice{display:none!important}main{display:block;padding:12px}.visual-workspace{display:block}#map-pane{width:100%}#map{height:500px;min-height:500px}</style></head>')
if args.suite:
 font=root/'panel/node_modules/@fontsource/inter/files/inter-latin-400-normal.woff2'
 if not font.exists():
  raise SystemExit('UI fixture needs its locked font: run just panel install')
 # Date is frozen but normal timers continue. advance() moves only fixture time.
 stable='''<script>
 const NativeDate=Date;let fixtureNow=Date.parse('2026-10-30T19:00:00Z');
 window.Date=class extends NativeDate {constructor(...a){super(...(a.length?a:[fixtureNow]));}static now(){return fixtureNow;}};
 window.renkiFixtureAdvance=ms=>{fixtureNow+=ms;};
 let fixtureSeed=102;Math.random=()=>{fixtureSeed=(1664525*fixtureSeed+1013904223)>>>0;return fixtureSeed/4294967296;};
 </script>'''
 css="@font-face{font-family:FixtureInter;src:url(data:font/woff2;base64,"+base64.b64encode(font.read_bytes()).decode()+");font-weight:100 900}*,*::before,*::after{animation:none!important;transition:none!important;caret-color:transparent!important;scroll-behavior:auto!important}body,button,input,select,textarea{font-family:FixtureInter,sans-serif!important}.preview-notice{position:fixed;top:0;left:0;right:0;z-index:10000;text-align:center;height:44px;display:flex;align-items:center;justify-content:center;box-sizing:border-box;padding:4px 12px!important;font-size:11px!important;line-height:14px}body{padding-top:44px}[id]{scroll-margin-top:154px!important}@media(min-width:761px){.topbar{top:44px!important}}"
 html=html.replace('<head>','<head>'+stable+'<style>'+css+'</style>')
 html=html.replace("connect-src \'self\';", "connect-src \'self\'; font-src data:;")
 html=html.replace('Recorded game assets · read-only fixture replay (not live)','FIXTURE · NOT LIVE · recorded map/JPEG + simulated events and timing')
out=pathlib.Path(args.output) if args.output else root/'.checks/map-ui-preview.html'
out.parent.mkdir(parents=True,exist_ok=True)
out.write_text(html)
print(out,flush=True)
if args.serve:
 class Handler(http.server.BaseHTTPRequestHandler):
  def do_GET(self):
   if self.path.split('?')[0] != '/':
    self.send_error(404);return
   data=out.read_bytes()
   self.send_response(200)
   self.send_header('Content-Type','text/html; charset=utf-8')
   self.send_header('Content-Length',str(len(data)))
   self.end_headers();self.wfile.write(data)
  def log_message(self,*_):pass
 print(f'http://127.0.0.1:{args.port}/',flush=True)
 http.server.HTTPServer(('127.0.0.1',args.port),Handler).serve_forever()
