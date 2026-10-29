"""Build a read-only preview of the real panel assets, optionally serving it.

Default terrain is a fixture. --status/--map use captured API snapshots.
No credentials, game tasks, external requests or third-party dependencies.
"""
import argparse
import http.server
import json,pathlib,math

parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument('--status')
parser.add_argument('--map')
parser.add_argument('--serve', action='store_true')
parser.add_argument('--port', type=int, default=18780)
args = parser.parse_args()
if bool(args.status) != bool(args.map):
 parser.error('--status and --map must be supplied together')
root=pathlib.Path(__file__).resolve().parent.parent
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
frames=[dict(type='layout',walls=[dict(a=[-42,36,-2],b=[-34,36,-5],h=2),dict(a=[-34,36,-5],b=[-27,36,13],h=2),dict(a=[-27,36,13],b=[-35,36,16],h=2),dict(a=[-35,36,16],b=[-42,36,-2],h=2)]),dict(type='perception',structures=[dict(id='chest',kind='container',pos=[-34,36,4]),dict(id='door',kind='door',open=True,pos=[-40,36,3]),dict(id='bench',kind='station',pos=[-30,36,9]),dict(id='bed',kind='bed',pos=[-34,36,12])],resources=[dict(id='tree',kind='tree',prefab='Beech1',pos=[-53,33,25])])]
mapped['walls'] = frames[0]['walls']
mapped['perception'] = frames[1]
mapped['containers'] = [dict(entity='chest',pos=[-34,36,4],size=[5,2],at='2026-10-29T18:00:00Z',items=[dict(prefab='CookedDeerMeat',stack=9),dict(prefab='Wood',stack=12)]),dict(entity='empty-chest',pos=[-54,33,4],size=[5,2],at='2026-10-29T17:00:00Z',items=[])]
frames[1]['structures'].extend([dict(id='empty-chest',kind='container',pos=[-54,33,4]),dict(id='unknown-chest',kind='container',pos=[-54,33,14])])
if args.status:
 status=json.loads(pathlib.Path(args.status).read_text())
 mapped=json.loads(pathlib.Path(args.map).read_text())
 frames=[]
html=(root/'internal/panel/static/index.html').read_text().replace('<link rel="stylesheet" href="/style.css">','<style>'+(root/'internal/panel/static/style.css').read_text()+'</style>')
def embedded(value):
 return json.dumps(value).replace('<', '\\u003c')

bootstrap='''<script>const previewStatus=STATUS, previewMap=MAPPED, previewFrames=FRAMES;
window.fetch=async(path,options={})=>options.method==='POST'?({ok:false,status:405,json:async()=>({text:'Read-only preview'})}):({ok:true,status:200,json:async()=>path==='/api/map'?previewMap:path==='/api/status'?previewStatus:[]});
window.EventSource=class{constructor(){this.handlers={};setTimeout(()=>{this.onopen?.();this.handlers.status?.({data:JSON.stringify(previewStatus)});for(const frame of previewFrames)this.handlers.record?.({data:JSON.stringify({kind:'frame',cat:'perception',frame})});},20)}addEventListener(n,fn){this.handlers[n]=fn}};</script>'''.replace('STATUS',embedded(status)).replace('MAPPED',embedded(mapped)).replace('FRAMES',embedded(frames))
html=html.replace('<script src="/terrain.js" defer></script>',bootstrap+'<script>'+(root/'internal/panel/static/terrain.js').read_text()+'</script>').replace('<script src="/app.js" defer></script>','')
notice='Recorded game snapshot · read-only preview' if args.status else 'Fixture terrain · read-only preview'
html=html.replace('<body>', '<body><div style="padding:5px 12px;color:#c3cbd1;background:#273039;font-size:12px">'+notice+'</div>')
checks='''<script>setTimeout(()=>{console.log(JSON.stringify({terrain:world.tiles.size,landmarks:world.landmarks.length,biome:document.getElementById('biome').textContent,surface:document.getElementById('surface').textContent,height:document.getElementById('height').textContent,errors:window.previewErrors||[]}));const first=document.querySelector('[data-layer=terrain]');first.click();if(layers.terrain!==false)throw Error('Terrain toggle failed');first.click();if(layers.terrain!==true)throw Error('Terrain toggle restore failed');console.log('Layer controls passed');},80);</script>'''
html=html.replace('</body>','<script>window.previewErrors=[];window.addEventListener("error",e=>previewErrors.push(e.message));</script><script>'+(root/'internal/panel/static/app.js').read_text()+'</script>'+checks+'</body>')
out=root/'.checks/map-ui-preview.html'
out.parent.mkdir(exist_ok=True)
out.write_text(html)
print(out, flush=True)
if args.serve:
 class Handler(http.server.BaseHTTPRequestHandler):
  def do_GET(self):
   if self.path != '/':
    self.send_error(404); return
   data=out.read_bytes()
   self.send_response(200)
   self.send_header('Content-Type','text/html; charset=utf-8')
   self.send_header('Content-Length',str(len(data)))
   self.end_headers(); self.wfile.write(data)
  def log_message(self, *_): pass
 print(f'http://127.0.0.1:{args.port}/', flush=True)
 http.server.HTTPServer(('127.0.0.1',args.port),Handler).serve_forever()
