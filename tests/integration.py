import json, threading, urllib.request, subprocess, time, pathlib, yaml, shutil
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
root=pathlib.Path('/home/frank/cliproxyapi-headroom'); work=root/'work'/'integration';work.mkdir(parents=True,exist_ok=True)
captures=[]
class Handler(BaseHTTPRequestHandler):
 def log_message(self,*a):pass
 def do_POST(self):
  body=json.loads(self.rfile.read(int(self.headers['Content-Length'])));captures.append(body)
  self.send_response(200)
  if body.get('stream'):
   self.send_header('Content-Type','text/event-stream');self.end_headers()
   for chunk in [{'id':'test','object':'chat.completion.chunk','model':'headroom-test','choices':[{'index':0,'delta':{'role':'assistant','content':'OK'},'finish_reason':None}]},{'id':'test','object':'chat.completion.chunk','model':'headroom-test','choices':[{'index':0,'delta':{},'finish_reason':'stop'}]}]:self.wfile.write(('data: '+json.dumps(chunk)+'\n\n').encode())
   self.wfile.write(b'data: [DONE]\n\n')
  else:
   self.send_header('Content-Type','application/json');self.end_headers();self.wfile.write(json.dumps({'id':'test','object':'chat.completion','model':'headroom-test','choices':[{'index':0,'message':{'role':'assistant','content':'OK'},'finish_reason':'stop'}],'usage':{'prompt_tokens':1,'completion_tokens':1,'total_tokens':2}}).encode())
server=ThreadingHTTPServer(('127.0.0.1',18319),Handler);threading.Thread(target=server.serve_forever,daemon=True).start()
config={'host':'127.0.0.1','port':18318,'auth-dir':'/tmp/test-auth','api-keys':['test-only'],'remote-management':{'secret-key':'headroom-integration-only','allow-remote':True},'plugins':{'enabled':True,'dir':'/plugins','configs':{'headroom':{'enabled':True,'endpoint':'http://127.0.0.1:8787/v1/compress','timeout_ms':60000,'stats_path':'/stats/stats.json'}}},'openai-compatibility':[{'name':'mock','base-url':'http://127.0.0.1:18319/v1','api-key-entries':[{'api-key':'mock-only'}],'models':[{'name':'headroom-test','alias':'headroom-test'}]}]}
(work/'config.yaml').write_text(yaml.safe_dump(config))
plugin=work/'plugins'/'linux'/'amd64';plugin.mkdir(parents=True,exist_ok=True);shutil.copy2(root/'dist/headroom-v0.3.0.so',plugin)
statsdir=work/'stats-v02';statsdir.mkdir(exist_ok=True)
name='headroom-plugin-integration'
subprocess.run(['docker','run','-d','--rm','--name',name,'--network','host','-v',str(work/'config.yaml')+':/CLIProxyAPI/config.yaml:ro','-v',str(work/'plugins')+':/plugins:ro','-v',str(statsdir)+':/stats','eceasy/cli-proxy-api:latest'],check=True,stdout=subprocess.DEVNULL)
try:
 for _ in range(60):
  try:
   req=urllib.request.Request('http://127.0.0.1:18318/v1/models',headers={'Authorization':'Bearer test-only'});urllib.request.urlopen(req,timeout=2).read();break
  except Exception:time.sleep(.5)
 log='2026-09-16 INFO health check succeeded status=200 service=api latency_ms=2\n'*500+'2026-09-16 ERROR payment failed transaction=TX-123 reason=insufficient_funds\n'
 summary=[]
 for stream in [False,True]:
  body={'model':'headroom-test','stream':stream,'messages':[{'role':'system','content':'Keep instructions intact'},{'role':'user','content':'Inspect the log'},{'role':'assistant','content':None,'tool_calls':[{'id':'call-real','type':'function','function':{'name':'read_log','arguments':'{}'}}]},{'role':'tool','tool_call_id':'call-real','content':log},{'role':'user','content':'What error occurred?'}]}
  req=urllib.request.Request('http://127.0.0.1:18318/v1/chat/completions',data=json.dumps(body).encode(),headers={'Authorization':'Bearer test-only','Content-Type':'application/json'})
  response=urllib.request.urlopen(req,timeout=90).read()
  got=captures[-1];out=got['messages'][3]['content']
  assert len(out)<len(log),(len(out),len(log))
  assert 'TX-123' in out and got['messages'][3]['tool_call_id']=='call-real'
  assert got['messages'][0]==body['messages'][0]
  assert b'OK' in response
  if stream:assert b'[DONE]' in response
  summary.append({'stream':stream,'tool_bytes_before':len(log),'tool_bytes_after':len(out),'unique_error_preserved':True,'provider_response_ok':True})

 def getstats():
  req=urllib.request.Request('http://127.0.0.1:18318/v0/management/plugins/headroom/stats',headers={'Authorization':'Bearer headroom-integration-only'})
  return json.load(urllib.request.urlopen(req,timeout=5))
 stats=getstats();assert stats['totals']['compressed']>=2,stats
 try:
  urllib.request.urlopen('http://127.0.0.1:18318/v0/management/plugins/headroom/stats',timeout=5)
  raise AssertionError('stats exposed without authentication')
 except urllib.error.HTTPError as e:assert e.code in (401,403)
 page=urllib.request.urlopen('http://127.0.0.1:18318/v0/resource/plugins/headroom/stats').read()
 assert b'Headroom compression' in page and b'TX-123' not in page
 req=urllib.request.Request('http://127.0.0.1:18318/v0/management/plugins',headers={'Authorization':'Bearer headroom-integration-only'})
 assert b'Headroom Stats' in urllib.request.urlopen(req).read()
 req=urllib.request.Request('http://127.0.0.1:18318/v0/management/plugins/headroom/service-stats',headers={'Authorization':'Bearer headroom-integration-only'})
 service=json.load(urllib.request.urlopen(req,timeout=10));assert len(service['endpoints'])==5 and all(e['ok'] for e in service['endpoints']),service
 time.sleep(3)
 subprocess.run(['docker','restart','-t','10',name],check=True,stdout=subprocess.DEVNULL)
 for _ in range(60):
  try:after=getstats();break
  except Exception:time.sleep(.5)
 assert after['totals']==stats['totals'],'stats lost on restart'
 summary.append({'stats_authenticated':True,'menu_registered':True,'restart_persistence':True,'compressed_requests':stats['totals']['compressed']})
 print(json.dumps(summary,indent=2));(work/'result.json').write_text(json.dumps(summary,indent=2))
finally:
 logs=subprocess.run(['docker','logs',name],capture_output=True,text=True);(work/'container.log').write_text(logs.stdout+logs.stderr)
 print('\n'.join(x for x in (logs.stdout+logs.stderr).splitlines() if 'headroom' in x.lower()))
 subprocess.run(['docker','stop','-t','2',name],stdout=subprocess.DEVNULL);server.shutdown()
