const {test,expect}=require('@playwright/test');
const fs=require('node:fs'),path=require('node:path'),os=require('node:os'),crypto=require('node:crypto');
const {execFileSync}=require('node:child_process');
const smoke=process.env.BRAWL_INPUT_SMOKE==='1';
const network={latency:150,downloadThroughput:200000,uploadThroughput:93750};
const percentile=(values,p)=>[...values].sort((a,b)=>a-b)[Math.ceil(values.length*p)-1]??null;
const git=(...args)=>execFileSync('git',args,{encoding:'utf8',maxBuffer:20*1024*1024}).trim();
const actions={right:{key:'d',field:'x',value:1},left:{key:'a',field:'x',value:-1},down:{key:'s',field:'y',value:1},up:{key:'w',field:'y',value:-1},jump:{key:'Space',field:'jump',value:true},dodge:{key:'c',field:'dodge',value:true}};
for(let sample=1;sample<=(smoke?1:3);sample++)test('isolated input confirmation sample '+sample,async({page,context,browser},info)=>{
 const report={sample,smoke,startedAt:new Date().toISOString(),revision:git('rev-parse','HEAD'),trackedDiffSHA256:crypto.createHash('sha256').update(git('diff','HEAD','--binary')).digest('hex'),browser:browser.version(),host:{cpu:os.cpus()[0]?.model,logicalCPUs:os.cpus().length,totalRAM:os.totalmem(),platform:os.platform()},profile:{width:1280,height:900,dpr:1,cpuSlowdown:4,preset:'lowPower',physicalMinimumDevice:false},network,thresholds:{p95:300,max:1000,minAccepted:50},scenario:'/abyss/rift?practice=touch&riftInputDebug=1',attempts:[],errors:[],status:'incomplete',releaseReady:false,scope:'Isolated keyboard movement, jump and dodge in free practice with warm art. Does not establish rapid-input, full combat or physical-device performance.'};
 const output=info.outputPath('input-report.json');fs.mkdirSync(path.dirname(output),{recursive:true});
 const save=()=>fs.writeFileSync(output,JSON.stringify(report,null,2)+'\n');
 const read=async()=>{const r=await page.request.get('/api/abyss/rift?practice=touch');expect(r.ok()).toBe(true);return(await r.json()).run;};
 page.on('pageerror',()=>report.errors.push({kind:'page'}));
 page.on('console',m=>{if(m.type()==='error')report.errors.push({kind:'console'});});
 page.on('response',r=>{if(r.status()>=400)report.errors.push({kind:'http',status:r.status()});});
 page.on('requestfailed',()=>report.errors.push({kind:'request'}));
 try{
  const cdp=await context.newCDPSession(page);await cdp.send('Emulation.setCPUThrottlingRate',{rate:4});
  await page.goto(report.scenario);await expect(page.locator('#rift-start')).toBeEnabled({timeout:120000});
  await page.locator('.rift-settings > summary').click();await page.locator('#rift-display-preset').selectOption('lowPower');await page.locator('#rift-apply-preset').click();await page.locator('.rift-settings > summary').click();
  await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();await page.locator('#rift-canvas').focus();
  await page.keyboard.down('d');
  try{await expect.poll(async()=>(await read()).player.x,{timeout:15000}).toBeGreaterThanOrEqual(650);}finally{await page.keyboard.up('d');}
  await page.waitForTimeout(5000);
  const positioned=await read();report.startPosition={x:positioned.player.x,y:positioned.player.y};expect(positioned.player.x).toBeGreaterThanOrEqual(650);expect(positioned.player.x).toBeLessThan(950);
  report.settings=await page.evaluate(()=>JSON.parse(localStorage.getItem('riftDisplay')));expect(report.settings.fps).toBe(30);
  await page.evaluate(()=>{window.inputCaptureHealth={hidden:false,contextLost:false};document.addEventListener('visibilitychange',()=>{if(document.hidden)inputCaptureHealth.hidden=true;});document.querySelector('#rift-canvas').addEventListener('contextlost',()=>inputCaptureHealth.contextLost=true);});
  report.actualViewport=await page.evaluate(()=>({width:innerWidth,height:innerHeight,dpr:devicePixelRatio}));expect(report.actualViewport).toEqual({width:1280,height:900,dpr:1});
  await cdp.send('Network.enable');await cdp.send('Network.emulateNetworkConditions',{offline:false,...network,connectionType:'cellular4g'});
  for(let cycle=0;cycle<(smoke?2:10);cycle++)for(const action of [...(cycle%2?['left','right']:['right','left']),'down','up','jump','dodge']){
   const definition=actions[action];let before;
   await expect.poll(async()=>{before=await read();return before.status==='fighting'&&!before.paused&&before.player.jump===0&&!(before.skill_timers.jump>0)&&!(before.skill_timers.dodge_cooldown>0)&&!['recovery','dodge','defeat'].includes(before.player.pose);},{timeout:30000}).toBe(true);
   await page.waitForTimeout([0,17,53,91,149][report.attempts.length%5]);
   const previous=await page.evaluate(()=>RiftInputDiagnostics.count);
   const attempt={before:{x:before.player.x,y:before.player.y},index:report.attempts.length+1,action,received:false,accepted:false,timing:null};report.attempts.push(attempt);save();
   const response=page.waitForResponse(r=>{const body=r.request().postDataJSON();return new URL(r.url()).pathname==='/api/abyss/rift'&&body?.kind==='step'&&body.input[definition.field]===definition.value;},{timeout:15000});
   await page.keyboard.press(definition.key);
   const result=await response;attempt.received=true;expect(result.ok()).toBe(true);
   const body=result.request().postDataJSON(),data=await result.json(),after=data.run;
   attempt.after={x:after.player.x,y:after.player.y};
   expect(after.id).toBe(body.run_id);expect(after.revision).toBe(body.revision);
   attempt.accepted=action==='jump'?after.stats.jumps===before.stats.jumps+1:action==='dodge'?after.skill_timers.dodge_cooldown>0:(after.player[definition.field]-before.player[definition.field])*definition.value>0;
   await expect.poll(()=>page.evaluate(()=>RiftInputDiagnostics.count),{timeout:15000}).toBe(previous+1);
   attempt.timing=await page.evaluate(()=>RiftInputDiagnostics.samples.at(-1));expect(attempt.timing.actions).toEqual([action]);save();
  }
  report.health=await page.evaluate(()=>({...inputCaptureHealth,visibility:document.visibilityState}));
  const timings=report.attempts.map(a=>a.timing.total);
  report.summary={count:timings.length,accepted:report.attempts.filter(a=>a.accepted).length,unconfirmed:report.attempts.filter(a=>!a.timing).length,rejected:report.attempts.filter(a=>a.received&&!a.accepted).length,p95:percentile(timings,.95),max:Math.max(...timings),queueP95:percentile(report.attempts.map(a=>a.timing.queue),.95),requestP95:percentile(report.attempts.map(a=>a.timing.request),.95)};
  expect(report.errors).toEqual([]);expect(report.health).toEqual({hidden:false,contextLost:false,visibility:'visible'});
  const s=report.summary;report.status=smoke?(s.accepted===12&&s.unconfirmed===0?'smoke_only':'capture_failure'):s.accepted>=50&&s.rejected===0&&s.unconfirmed===0&&s.p95<=300&&s.max<=1000?'isolated_input_numeric_pass':'threshold_failure';
  console.log(JSON.stringify({sample,status:report.status,...s}));
 }catch(_){report.status='capture_failure';throw _;}finally{report.finishedAt=new Date().toISOString();save();}
});
