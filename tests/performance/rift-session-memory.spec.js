const {test,expect}=require('@playwright/test');
const fs=require('node:fs');
const os=require('node:os');
const crypto=require('node:crypto');
const {execFileSync}=require('node:child_process');
const smoke=process.env.BRAWL_SESSION_SMOKE==='1';
const durationMS=smoke?60000:30*60*1000;
const git=(...args)=>execFileSync('git',args,{encoding:'utf8',maxBuffer:20*1024*1024}).trim();

const {summarizeHeap}=require('../../scripts/brawl-heap-summary.cjs');
const {createLedgeNavigator}=require('../../scripts/brawl-session-navigation.cjs');

for(let sample=1;sample<=(smoke?1:3);sample++)test(`campaign session memory sample ${sample}`,async({page,context,browser},info)=>{
 const path=require('node:path'),output=info.outputPath('memory-report.json');
 fs.mkdirSync(path.dirname(output),{recursive:true});
 const sourceFile=path.join(path.dirname(path.dirname(output)),'fixture-source.json');
 if(!fs.existsSync(sourceFile)){
  const patch=git('diff','HEAD','--binary');
  fs.writeFileSync(sourceFile,JSON.stringify({revision:git('rev-parse','HEAD'),trackedDiffSHA256:crypto.createHash('sha256').update(patch).digest('hex'),dirtyFiles:git('status','--short'),patch,capturedAt:new Date().toISOString(),scope:'Checkout captured by first sample immediately after fresh shared fixture startup; no rebuild between samples.'},null,2)+'\n',{flag:'wx'});
 }
 const source=JSON.parse(fs.readFileSync(sourceFile,'utf8'));
 const report={startedAt:new Date().toISOString(),mode:smoke?'smoke (not a gate run)':'30 minutes of repeated complete three-tier missions',sample,durationMS,
  server:{command:'go test -tags=e2e ./internal/bot -run TestAbyssE2EServer -count=1 -v -timeout=130m',managedFresh:true,revision:source.revision,trackedDiffSHA256:source.trackedDiffSHA256,dirtyFiles:source.dirtyFiles,sourceCapturedAt:source.capturedAt},
  host:{platform:os.platform(),release:os.release(),cpu:os.cpus()[0]?.model,logicalCPUs:os.cpus().length,totalRAM:os.totalmem(),freeRAM:os.freemem()},
  browser:browser.version(),profile:{viewport:{width:1280,height:900},dpr:1,cpuSlowdown:4,display:'lowPower',physicalMinimumDevice:false},
  scenario:'/abyss/rift?subclass=bloodblade&riftFrameDebug=1',seed:'production random; replay seeds and run IDs captured per cycle',mission:1,
  protocol:'At completed-mission boundaries after each five-minute interval: settle 2s, GC, settle 1s, GC, heap and DOM counters plus heap snapshot. Record actual elapsed time; no reloads.',
  checkpoints:[],expeditions:[],errors:[],gate:'unmeasured',processMemory:'Windows working set and private bytes for CDP-listed browser processes; kept separate from JS heap'};
 const save=()=>fs.writeFileSync(output,JSON.stringify(report,null,2)+'\n');
 fs.writeFileSync(info.outputPath('server-tracked-diff.patch'),source.patch);
 report.command='node node_modules/@playwright/test/cli.js test --config=playwright.session-memory.config.js';
 const session=await context.newCDPSession(page);
 const browserSession=await browser.newBrowserCDPSession();
 let sampleStarted=0;
 const held=new Set();
 async function controls(wanted){
  for(const key of [...held])if(!wanted.has(key)){await page.keyboard.up(key);held.delete(key);}
  for(const key of wanted)if(!held.has(key)){await page.keyboard.down(key);held.add(key);}
 }
 const read=async()=>{const response=await page.request.get('/api/abyss/rift');expect(response.ok()).toBe(true);return(await response.json()).run;};
 async function expedition(index){
  const started=Date.now();const existing=await read();
  await page.locator(existing?.status==='complete'?'#rift-replay':'#rift-start').click();
  await expect(page.locator('#rift-overlay')).toBeHidden();
  let run;const tiers=[];let lastInputReset=0;const waypoint=createLedgeNavigator();
  try{
   while(Date.now()-started<360000){
    if(report.errors.length)throw Error('Runtime failure during campaign: '+report.errors[0].message);
    run=await read();report.lastCombat={room:run.room,status:run.status,player:{x:run.player.x,y:run.player.y,hp:run.player.hp},enemies:run.enemies.filter(e=>e.hp>0).map(e=>({kind:e.kind,x:e.x,y:e.y,hp:e.hp}))};expect(run.status,'combat must reach a checkpoint without death').not.toBe('defeated');
    if(run.status==='cleared'){
     await controls(new Set());tiers.push({room:run.room,clock:run.clock,kills:run.stats.kills});
     await expect(page.locator('#rift-next')).toBeVisible();await page.locator('#rift-next').click();
     const previous=run.room;
     await expect.poll(async()=>{const next=await read();return next.room!==previous||next.status==='complete';}).toBe(true);
     run=await read();if(run.status==='complete')break;
     await expect(page.locator('#rift-next')).toBeHidden();
     if(run.paused){await page.locator('#rift-start').click();}
     continue;
    }
    expect(run.status).toBe('fighting');
    const p=run.player,target=run.enemies.filter(e=>e.hp>0).sort((a,b)=>Math.abs(a.x-p.x)-Math.abs(b.x-p.x))[0];
    const wanted=new Set(['Space']);
    if(target){
     const dx=target.x-p.x,dy=target.y-p.y;
     const destination=waypoint(run,target),mx=destination.x-p.x,my=destination.y-p.y,detouring=destination!==target;
     if(Math.abs(my)>(detouring?5:10))wanted.add(my>0?'s':'w');
     if(Math.abs(mx)>(detouring?6:60)||!detouring&&Math.sign(dx)!==p.facing)wanted.add(mx>0?'d':'a');
     if(Math.abs(dx)<120&&Math.abs(dy)<30){
      const [builder,finisher]=run.build.signatures;
      if(run.resource>0&&!(run.skill_timers[finisher.id]>0)&&p.mana>=finisher.cost)wanted.add('e');
      else if(!(run.skill_timers[builder.id]>0)&&p.mana>=builder.cost)wanted.add('q');
      else wanted.add('j');
     }
    }
    // A tier snapshot can clear held client input; periodically release and
    // repress keys like a human adjusting movement, without bypassing controls.
    if(Date.now()-lastInputReset>1500){await controls(new Set());lastInputReset=Date.now();}
    await controls(wanted);await page.waitForTimeout(120);
   }
  }finally{await controls(new Set());}
  expect(run?.status).toBe('complete');expect(tiers.map(t=>t.room)).toEqual([0,1,2]);
  await expect(page.locator('#rift-overlay')).toBeVisible();
  report.expeditions.push({index,id:run.id,replaySeed:run.replay_seed,tiers,status:run.status,durationMS:Date.now()-started});save();
 }
 async function checkpoint(index,snapshot){
  await page.waitForTimeout(2000);await session.send('HeapProfiler.collectGarbage');
  await page.waitForTimeout(1000);await session.send('HeapProfiler.collectGarbage');
  const point={mission:index,elapsedMS:sampleStarted?Date.now()-sampleStarted:0,at:new Date().toISOString(),heap:await session.send('Runtime.getHeapUsage'),dom:await session.send('Memory.getDOMCounters')};
  const processes=(await browserSession.send('SystemInfo.getProcessInfo')).processInfo;
  const ids=processes.map(p=>p.id);expect(ids.every(id=>Number.isSafeInteger(id)&&id>0)).toBe(true);
  const memory=JSON.parse(execFileSync('powershell.exe',['-NoProfile','-Command',
   'Get-Process -Id '+ids.join(',')+' -ErrorAction SilentlyContinue | Select-Object Id,WorkingSet64,PrivateMemorySize64 | ConvertTo-Json -Compress'],{encoding:'utf8'}));
  point.browserProcesses=(Array.isArray(memory)?memory:[memory]).map(p=>({type:processes.find(v=>v.id===p.Id)?.type,pid:p.Id,workingSet:p.WorkingSet64,privateBytes:p.PrivateMemorySize64}));
  point.render=await page.evaluate(()=>({...sessionHealth,frameAgeMS:performance.now()-sessionHealth.lastFrame}));
  expect(point.render.frames).toBeGreaterThan(0);expect(point.render.frameAgeMS).toBeLessThan(5000);expect(point.render.hidden).toBe(false);expect(point.render.contextLost).toBe(false);
  point.audio=await page.evaluate(()=>({voices:window.RiftAudio.voices,context:window.RiftAudio.context?.state,visible:document.visibilityState}));
  expect(point.audio.visible).toBe('visible');expect(point.audio.voices).toBe(0);
  if(snapshot){
   point.snapshot=info.outputPath(`heap-${index}.heapsnapshot`);
   const fd=fs.openSync(point.snapshot,'w');const chunk=event=>fs.writeSync(fd,event.chunk);
   session.on('HeapProfiler.addHeapSnapshotChunk',chunk);
   try{await session.send('HeapProfiler.takeHeapSnapshot',{reportProgress:false});}finally{session.off('HeapProfiler.addHeapSnapshotChunk',chunk);fs.closeSync(fd);}
   point.reachable=summarizeHeap(JSON.parse(fs.readFileSync(point.snapshot,'utf8')));
  }
  report.checkpoints.push(point);save();console.log(`Memory sample ${sample}, mission ${index}: ${(point.heap.usedSize/1048576).toFixed(2)} MiB`);
 }
 page.on('console',message=>{if(message.type()==='error')report.errors.push({kind:'console',message:message.text()});});
 page.on('pageerror',error=>{report.errors.push({kind:'page',message:error.message,stack:error.stack,at:new Date().toISOString()});save();});
 page.on('response',response=>{if(response.status()>=400)report.errors.push({kind:'http',path:new URL(response.url()).pathname,status:response.status()});});
 try{
  await session.send('Emulation.setCPUThrottlingRate',{rate:4});
  await page.goto(report.scenario);report.server.assetBuild=(await(await page.request.get('/api/abyss/rift')).json()).build;await expect(page.locator('#rift-start')).toBeEnabled({timeout:120000});
  await page.locator('#rift-auto').uncheck();
  await page.locator('.rift-settings > summary').click();
  await page.locator('#rift-display-preset').selectOption('lowPower');
  await page.locator('#rift-apply-preset').click();
  await page.locator('.rift-settings > summary').click();
  report.settings=await page.evaluate(()=>({display:JSON.parse(localStorage.getItem('riftDisplay')),audio:localStorage.getItem('riftAudio'),visibility:document.visibilityState}));
  expect(report.settings.display.fps).toBe(30);expect(report.settings.display.particles).toBe(false);
  report.graphics=(await browserSession.send('SystemInfo.getInfo')).gpu;
  await page.evaluate(()=>{
   const d=RiftRenderer.frameDiagnostics,original=d.samples.push;
   const health=window.sessionHealth={frames:0,lastFrame:0,hidden:false,contextLost:false};
   d.samples.push=function(...values){health.frames+=values.length;health.lastFrame=performance.now();return original.apply(this,values);};
   document.addEventListener('visibilitychange',()=>{if(document.visibilityState!=='visible')health.hidden=true;});
   document.querySelector('#rift-canvas').addEventListener('contextlost',()=>health.contextLost=true);
  });
  // Warm all three tiers and their boss before the completed-mission baseline.
  await expedition('warmup');await page.waitForTimeout(5000);await checkpoint(0,true);
  sampleStarted=Date.now();let index=0,nextCheckpoint=smoke?durationMS:5*60*1000;
  while(Date.now()-sampleStarted<durationMS){
   await expedition(++index);
   if(Date.now()-sampleStarted>=nextCheckpoint){await checkpoint(index,true);nextCheckpoint=Date.now()-sampleStarted+5*60*1000;}
  }
  if(report.checkpoints.at(-1).mission!==index)await checkpoint(index,true);
  report.measuredDurationMS=Date.now()-sampleStarted;
  expect(report.measuredDurationMS).toBeGreaterThanOrEqual(durationMS);
  expect(new Set(report.expeditions.map(e=>e.id)).size).toBe(index+1);
  const first=report.checkpoints[0],last=report.checkpoints.at(-1);
  report.growthBytes=last.heap.usedSize-first.heap.usedSize;
  report.heapSizeGate=report.growthBytes<=10*1048576?'pass':'fail';
  const final=report.checkpoints.slice(-5);
  report.finalFive=final.map(p=>({mission:p.mission,usedSize:p.heap.usedSize,...p.dom}));
  report.gate=smoke?'unmeasured (smoke only)':report.heapSizeGate==='fail'||report.errors.length?'fail':'retaining-path review required';
  expect(report.errors).toEqual([]);expect(report.growthBytes).toBeLessThanOrEqual(10*1048576);
 }catch(error){report.error=error.message;report.gate=report.errors.length?'fail (runtime error)':report.gate==='fail'?'fail':'incomplete';throw error;}
 finally{report.finishedAt=new Date().toISOString();save();await session.detach();await browserSession.detach();}
});
