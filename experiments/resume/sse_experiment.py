#!/usr/bin/env python3
"""Isolated, real-image SSE experiment. Run only on ephemeral Actions runner."""
import asyncio, json, os, pathlib, re, subprocess, time, uuid
import aiohttp
from aiohttp import web

OUT=pathlib.Path(os.environ.get('EXPERIMENT_OUT','results'));OUT.mkdir(exist_ok=True,parents=True)
IMAGE='ghcr.io/floating0516/sub2api-tocreate@sha256:1929e4a9744a540fb3e2cba45acba2dc3c193af4d1b978cf632e92f958be0f3c'
PASSWORD='Synthetic-Experiment-Only-20261001!'
STATES={}; ROWS=[]; ACCOUNTS=[]; TOKEN=None

def cmd(*args):
    return subprocess.check_output(args,text=True,stderr=subprocess.STDOUT).strip()
def sql(query):
    return cmd('docker','exec','resume-pg','psql','-U','postgres','-d','resume','-At','-c',query)
def docker_start(name,image,*options):
    return cmd('docker','run','-d','--name',name,'--network','host',*options,image)
def event(kind,**data):
    return ('event: '+kind+'\ndata: '+json.dumps({'type':kind,**data})+'\n\n').encode()

async def upstream(request):
    raw=await request.text()
    match=re.search(r'EXP:([a-z0-9]+):(headers|body):([0-9]+):(ok|error|cancel)',raw)
    if not match:
        return web.json_response({'error':{'message':'missing experiment marker','type':'invalid_request_error'}},status=400)
    rid,phase,delay,mode=match.groups();delay=int(delay)
    state={'id':rid,'phase':phase,'delay_s':delay,'mode':mode,'path':request.path,'started':time.monotonic(),'state':'waiting'}
    STATES[rid]=state
    response=web.StreamResponse(headers={'Content-Type':'text/event-stream','Cache-Control':'no-cache'})
    transport=request.transport
    try:
        if phase=='body':
            await response.prepare(request)
            state['headers_flushed']=True
        until=time.monotonic()+delay
        while time.monotonic()<until:
            if transport is None or transport.is_closing():
                state.update(state='cancelled',ended=time.monotonic());return response
            await asyncio.sleep(.05)
        if mode=='error':
            if phase=='headers':
                state.update(state='error',ended=time.monotonic())
                return web.json_response({'error':{'message':'synthetic upstream failure '+rid,'type':'invalid_request_error','code':'synthetic_error'}},status=400)
            await response.write(event('error',message='synthetic upstream failure '+rid,code='synthetic_error'))
        else:
            if phase=='headers':await response.prepare(request)
            response_id='resp_'+rid
            await response.write(event('response.created',response={'id':response_id,'object':'response','status':'in_progress','model':'gpt-5','output':[]}))
            await response.write(event('response.output_item.added',output_index=0,item={'id':'msg_'+rid,'type':'message','role':'assistant','status':'in_progress','content':[]}))
            await response.write(event('response.content_part.added',item_id='msg_'+rid,output_index=0,content_index=0,part={'type':'output_text','text':'','annotations':[]}))
            await response.write(event('response.output_text.delta',item_id='msg_'+rid,output_index=0,content_index=0,delta='EXPERIMENT_OK'))
            await response.write(event('response.output_text.done',item_id='msg_'+rid,output_index=0,content_index=0,text='EXPERIMENT_OK'))
            await response.write(event('response.completed',response={'id':response_id,'object':'response','model':'gpt-5','status':'completed','output':[{'type':'message','id':'msg_'+rid,'role':'assistant','status':'completed','content':[{'type':'output_text','text':'EXPERIMENT_OK','annotations':[]}]}],'usage':{'input_tokens':5,'output_tokens':2,'total_tokens':7}}))
        await response.write_eof();state.update(state='completed' if mode=='ok' else 'error',ended=time.monotonic())
    except (ConnectionError,RuntimeError,asyncio.CancelledError):
        state.update(state='cancelled',ended=time.monotonic())
    return response

async def api(session,port,path,payload=None,token=None):
    headers={'Authorization':'Bearer '+token} if token else {}
    async with session.request('POST' if payload is not None else 'GET',f'http://127.0.0.1:{port}/api/v1'+path,json=payload,headers=headers) as r:
        text=await r.text()
        if r.status>=400:raise RuntimeError(f'API {path}: {r.status} {text[:700]}')
        obj=json.loads(text)
        if isinstance(obj,dict) and obj.get('code',0)!=0:raise RuntimeError(f'API {path}: {text[:700]}')
        return obj.get('data',obj)

async def wait_ready(session,port):
    for _ in range(120):
        try:
            async with session.get(f'http://127.0.0.1:{port}/ready') as r:
                if r.status==200:return
        except aiohttp.ClientError:pass
        await asyncio.sleep(1)
    raise RuntimeError(f'gateway {port} not ready')

def launch_gateway(label,port,interval):
    env={'AUTO_SETUP':'true','SERVER_HOST':'127.0.0.1','SERVER_PORT':str(port),'SERVER_MODE':'release','RUN_MODE':'standard','DATABASE_HOST':'127.0.0.1','DATABASE_PORT':'15439','DATABASE_USER':'postgres','DATABASE_PASSWORD':PASSWORD,'DATABASE_DBNAME':'resume','DATABASE_SSLMODE':'disable','REDIS_HOST':'127.0.0.1','REDIS_PORT':'16389','REDIS_DB':'0' if interval==0 else '1','ADMIN_EMAIL':'experiment@example.invalid','ADMIN_PASSWORD':PASSWORD,'JWT_SECRET':'synthetic-resume-experiment-jwt-secret-20261001','TZ':'Asia/Shanghai','GATEWAY_STREAM_KEEPALIVE_INTERVAL':str(interval),'GATEWAY_STREAM_DATA_INTERVAL_TIMEOUT':'60','GATEWAY_RESPONSE_HEADER_TIMEOUT':'60','GATEWAY_OPENAI_RESPONSE_HEADER_TIMEOUT':'60','SECURITY_URL_ALLOWLIST_ENABLED':'false','SECURITY_URL_ALLOWLIST_ALLOW_PRIVATE_HOSTS':'true','SECURITY_URL_ALLOWLIST_ALLOW_INSECURE_HTTP':'true','SERVER_BACKGROUND_START_DELAY_SECONDS':'3600'}
    args=[]
    for k,v in env.items():args+=['-e',k+'='+v]
    docker_start('resume-'+label,IMAGE,'--cpus','0.8','--memory','1200m','--log-opt','max-size=20m','--log-opt','max-file=3',*args)

async def one(session,key,keepalive,phase,delay,mode='ok',cancel_after=None):
    rid=uuid.uuid4().hex;start=time.monotonic();port=19081 if keepalive else 19080
    row={'id':rid,'keepalive_s':keepalive,'phase':phase,'delay_s':delay,'mode':mode,'status':None,'first_ping_s':None,'first_content_s':None,'complete':False,'error_event':False,'exception':None,'pings':0,'content':'','normal_end':False}
    response=None
    async def consume():
        nonlocal response
        async with session.post(f'http://127.0.0.1:{port}/v1/messages',json={'model':'gpt-5','stream':True,'max_tokens':64,'messages':[{'role':'user','content':f'EXP:{rid}:{phase}:{delay}:{mode}'}]},headers={'x-api-key':key,'anthropic-version':'2023-06-01','Content-Type':'application/json'},timeout=aiohttp.ClientTimeout(total=75)) as r:
            response=r;row['status']=r.status;row['headers_s']=time.monotonic()-start
            if r.status>=400:
                row['http_error_body']=(await r.text())[:1000]
                return
            async for line in r.content:
                line=line.decode(errors='replace').strip()
                if not line.startswith('data:'):continue
                try:obj=json.loads(line[5:])
                except json.JSONDecodeError:continue
                kind=obj.get('type')
                if kind=='ping':
                    row['pings']+=1
                    if row['first_ping_s'] is None:row['first_ping_s']=time.monotonic()-start
                if kind=='content_block_delta' and obj.get('delta',{}).get('type')=='text_delta':
                    if row['first_content_s'] is None:row['first_content_s']=time.monotonic()-start
                    row['content']+=obj['delta'].get('text','')
                if kind=='message_stop':row['normal_end']=True
                if kind=='error':row['error_event']=True;row['error_payload']=obj
    task=asyncio.create_task(consume())
    try:
        if cancel_after is not None:
            await asyncio.sleep(cancel_after)
            task.cancel()
            if response is not None:response.close()
            try:await task
            except asyncio.CancelledError:pass
            row['exception']='client_cancel'
        else:await task
    except Exception as exc:row['exception']=type(exc).__name__+': '+str(exc)[:180]
    row['duration_s']=time.monotonic()-start
    row['complete']=row['content']=='EXPERIMENT_OK' and row['normal_end'] and not row['error_event'] and row['exception'] is None
    if cancel_after is not None:
        for _ in range(60):
            if STATES.get(rid,{}).get('state')=='cancelled':break
            await asyncio.sleep(.05)
    state=STATES.get(rid,{})
    row['upstream_state']=state.get('state','not_reached');row['upstream_path']=state.get('path')
    if cancel_after is not None:
        row['cancel_propagated']=state.get('state')=='cancelled'
        if 'ended' in state:row['cancel_release_s']=state['ended']-(start+cancel_after)
    ROWS.append(row)
    with (OUT/'sse-requests.jsonl').open('a') as f:f.write(json.dumps(row)+'\n')
    return row

async def main():
    app=web.Application();app.router.add_post('/{tail:.*}',upstream)
    runner=web.AppRunner(app);await runner.setup();await web.TCPSite(runner,'127.0.0.1',19090).start()
    async with aiohttp.ClientSession() as session:
        launch_gateway('off',18090,0);await wait_ready(session,18090)
        login=await api(session,18090,'/auth/login',{'email':'experiment@example.invalid','password':PASSWORD});token=login['access_token']
        group=await api(session,18090,'/admin/groups',{'name':'resume-experiment','platform':'openai','subscription_type':'standard','rate_multiplier':1,'is_exclusive':False},token)
        account=await api(session,18090,'/admin/accounts',{'name':'synthetic-upstream','platform':'openai','type':'apikey','credentials':{'api_key':'synthetic-not-a-real-key','base_url':'http://127.0.0.1:19090/v1'},'group_ids':[group['id']],'concurrency':100,'priority':1,'rate_multiplier':1},token)
        sql("UPDATE users SET balance=10000, concurrency=100 WHERE email='experiment@example.invalid'")
        key=await api(session,18090,'/keys',{'name':'resume-experiment','group_id':group['id']},token);key=key['key']
        launch_gateway('on',18091,5);await wait_ready(session,18091)
        nginx=OUT/'nginx.conf';nginx.write_text('events { worker_connections 1024; }\nhttp { access_log /dev/stdout; error_log /dev/stderr info; '+''.join(f'server {{ listen {p}; location / {{ proxy_pass http://127.0.0.1:{g}; proxy_http_version 1.1; proxy_set_header Connection ""; proxy_buffering off; proxy_cache off; proxy_read_timeout 15s; proxy_send_timeout 60s; proxy_ignore_client_abort off; }} }}' for p,g in [(19080,18090),(19081,18091)])+'}\n')
        docker_start('resume-nginx','nginx:1.28-alpine','-v',str(nginx.resolve())+':/etc/nginx/nginx.conf:ro')
        await asyncio.sleep(2)
        probe=await one(session,key,5,'headers',0)
        (OUT/'sse-probe.json').write_text(json.dumps(probe,indent=2))
        if not probe['complete']:raise RuntimeError('Baseline protocol probe failed: '+json.dumps(probe))
        # Probe is excluded from measured request dataset.
        ROWS.clear();(OUT/'sse-requests.jsonl').write_text('')
        for phase in ['headers','body']:
            for delay in [10,17,30]:
                async def group(interval):
                    sem=asyncio.Semaphore(10)
                    async def sample():
                        async with sem:return await one(session,key,interval,phase,delay)
                    return await asyncio.gather(*(sample() for _ in range(50)))
                results=await asyncio.gather(group(0),group(5))
                print(json.dumps({'phase':phase,'delay_s':delay,'off_complete':sum(r['complete'] for r in results[0]),'on_complete':sum(r['complete'] for r in results[1]),'per_group':50}),flush=True)
        for interval in [0,5]:
            for _ in range(3):
                await one(session,key,interval,'headers',1,'error')
                await one(session,key,interval,'headers',7,'error')
                await one(session,key,interval,'headers',30,'cancel',cancel_after=7)
                await one(session,key,interval,'body',30,'cancel',cancel_after=7)
        normal=[r for r in ROWS if r['mode']=='ok']
        summary={'baseline_image':IMAGE,'simulation':True,'proxy_read_timeout_s':15,'heartbeat_s':[0,5],'client_timeout_s':75,'gateway_header_timeout_s':60,'gateway_stream_idle_timeout_s':60,'concurrency_per_arm':10,'paired_arms_simultaneous':True,'samples':len(normal),'cells':[],'extra_cases':[r for r in ROWS if r['mode']!='ok']}
        for phase in ['headers','body']:
            for delay in [10,17,30]:
                for interval in [0,5]:
                    rows=[r for r in normal if (r['phase'],r['delay_s'],r['keepalive_s'])==(phase,delay,interval)]
                    summary['cells'].append({'phase':phase,'delay_s':delay,'keepalive_s':interval,'n':len(rows),'complete':sum(r['complete'] for r in rows),'http_504':sum(r['status']==504 for r in rows),'stream_errors':sum(r['error_event'] for r in rows),'transport_exceptions':sum(r['exception'] is not None for r in rows),'first_content_s':[r['first_content_s'] for r in rows if r['first_content_s'] is not None]})
        (OUT/'sse-summary.json').write_text(json.dumps(summary,indent=2))
    await runner.cleanup()

if __name__=='__main__':
    try:asyncio.run(main())
    finally:
        for name in ['resume-off','resume-on','resume-nginx']:
            try:(OUT/(name+'.log')).write_text(cmd('docker','logs',name))
            except Exception as exc:(OUT/(name+'.log')).write_text(str(exc))
        (OUT/'mock-states.json').write_text(json.dumps(STATES,indent=2))
