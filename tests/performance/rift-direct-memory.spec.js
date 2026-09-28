const {test,expect,chromium}=require('@playwright/test');
const fs=require('node:fs'),path=require('node:path'),crypto=require('node:crypto');
const {execFileSync}=require('node:child_process');const {setTimeout:delay}=require('node:timers/promises');
const {launchDirectChromium}=require('../../scripts/brawl-direct-chromium.cjs');
const {attachDirectPage}=require('../../scripts/brawl-direct-page.cjs');
const {createLedgeNavigator}=require('../../scripts/brawl-session-navigation.cjs');
const {summarizeHeap}=require('../../scripts/brawl-heap-summary.cjs');
const smoke=process.env.BRAWL_DIRECT_MEMORY_SMOKE==='1',durationMS=smoke?60000:1800000;
const endpointSnapshots=process.env.BRAWL_DIRECT_SNAPSHOT_ENDPOINTS==='1';
const git=(...args)=>execFileSync('git',args,{encoding:'utf8',maxBuffer:20*1024*1024}).trim();
for(const networkInspection of [false,true])test('direct campaign network inspection '+networkInspection,async({},info)=>{
 const root=path.dirname(info.outputPath('memory-report.json'));fs.mkdirSync(root,{recursive:true});
 const sourcePath=path.join(path.dirname(root),'fixture-source.json');
 const driverFiles=['tests/performance/rift-direct-memory.spec.js','scripts/brawl-direct-page.cjs','scripts/brawl-direct-cdp.cjs','scripts/brawl-direct-chromium.cjs','scripts/brawl-session-navigation.cjs'];
 if(!fs.existsSync(sourcePath)){const patch=git('diff','HEAD','--binary');fs.writeFileSync(sourcePath,JSON.stringify({revision:git('rev-parse','HEAD'),trackedDiffSHA256:crypto.createHash('sha256').update(patch).digest('hex'),patch,drivers:driverFiles.map(file=>({file,source:fs.readFileSync(file,'utf8')}))},null,2)+'\n',{flag:'wx'});}
 const fixtureSource=JSON.parse(fs.readFileSync(sourcePath,'utf8')),patch=fixtureSource.patch;
 const report={sample:networkInspection?2:1,mode:smoke?'direct browser smoke; not a gate':endpointSnapshots?'direct browser endpoint-snapshot instrumentation diagnostic':'direct browser paired instrumentation diagnostic',heapSnapshotPolicy:endpointSnapshots?'endpoints':'all',networkInspection,startedAt:new Date().toISOString(),server:{revision:fixtureSource.revision,trackedDiffSHA256:fixtureSource.trackedDiffSHA256},checkpoints:[],expeditions:[],errors:[],releaseReady:false};
 report.command='node node_modules/@playwright/test/cli.js test --config=playwright.direct-memory.config.js';
 report.driver=fixtureSource.drivers.map(({file,source})=>{const data=Buffer.from(source);fs.writeFileSync(path.join(root,path.basename(file)+'.source'),data);return {file,sha256:crypto.createHash('sha256').update(data).digest('hex')};});
 const save=()=>fs.writeFileSync(path.join(root,'memory-report.json'),JSON.stringify(report,null,2)+'\n');fs.writeFileSync(path.join(root,'server-tracked-diff.patch'),patch);
 const browser=await launchDirectChromium({executablePath:chromium.executablePath(),outputDirectory:root});let page,started=0;
 try{
  report.browser=browser.version;report.launchArguments=browser.args;
  const url='http://127.0.0.1:'+(process.env.ABYSS_E2E_PORT||'18098')+'/abyss/rift?subclass=bloodblade&riftFrameDebug=1';
  page=await attachDirectPage(browser.cdp,{url,networkInspection});
  report.profile=await page.evaluate(()=>({width:innerWidth,height:innerHeight,dpr:devicePixelRatio,headless:true,physicalMinimumDevice:false,cpuSlowdown:4}));expect(report.profile.width).toBe(1280);expect(report.profile.height).toBe(900);expect(report.profile.dpr).toBe(1);
  if(await page.evaluate(()=>document.querySelector('#rift-auto').checked))await page.click('#rift-auto');
  await page.click('.rift-settings > summary');await page.evaluate(()=>{const select=document.querySelector('#rift-display-preset');select.value='lowPower';select.dispatchEvent(new Event('change',{bubbles:true}));});await page.click('#rift-apply-preset');await page.click('.rift-settings > summary');
  report.settings=await page.evaluate(()=>JSON.parse(localStorage.getItem('riftDisplay')));expect(report.settings.fps).toBe(30);expect(report.settings.particles).toBe(false);
  report.graphics=(await browser.cdp.send('SystemInfo.getInfo')).gpu;
  await page.evaluate(()=>{const d=RiftRenderer.frameDiagnostics,original=d.samples.push;window.directHealth={frames:0,lastFrame:0,hidden:false,contextLost:false};d.samples.push=function(...values){directHealth.frames+=values.length;directHealth.lastFrame=performance.now();return original.apply(this,values);};document.addEventListener('visibilitychange',()=>{if(document.visibilityState!=='visible')directHealth.hidden=true;});document.querySelector('#rift-canvas').addEventListener('contextlost',()=>directHealth.contextLost=true);});
  async function expedition(index){
   const began=Date.now(),existing=await page.read(),tiers=[],waypoint=createLedgeNavigator();let run,lastInputReset=0;
   await page.click(existing?.status==='complete'?'#rift-replay':'#rift-start');
   try{while(Date.now()-began<360000){
    await page.checkErrors();run=await page.read();expect(run.status).not.toBe('defeated');
    if(run.status==='cleared'){
     await page.controls(new Set());tiers.push({room:run.room,clock:run.clock,kills:run.stats.kills});await page.click('#rift-next');const previous=run.room;
     await page.wait(async()=>{const next=await page.read();return next.room!==previous||next.status==='complete';});run=await page.read();if(run.status==='complete')break;if(run.paused)await page.click('#rift-start');continue;
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

    if(Date.now()-lastInputReset>1500){await page.controls(new Set());lastInputReset=Date.now();}
    await page.controls(wanted);await delay(120);
   }}finally{await page.controls(new Set());}
   expect(run?.status).toBe('complete');expect(tiers.map(t=>t.room)).toEqual([0,1,2]);report.expeditions.push({index,id:run.id,replaySeed:run.replay_seed,tiers,status:run.status,durationMS:Date.now()-began});save();
  }
  async function checkpoint(mission,final=false){
   await delay(2000);await page.send('HeapProfiler.collectGarbage');await delay(1000);await page.send('HeapProfiler.collectGarbage');
   await page.checkErrors();const run=await page.read();const point={mission,elapsedMS:started?Date.now()-started:0,attemptHistoryCount:(run.attempt_history||[]).length,heap:await page.send('Runtime.getHeapUsage'),dom:await page.send('Memory.getDOMCounters'),networkEvents:page.networkEvents};
   const processes=(await browser.cdp.send('SystemInfo.getProcessInfo')).processInfo,ids=processes.map(p=>p.id);expect(ids.every(id=>Number.isSafeInteger(id)&&id>0)).toBe(true);
   const counters=JSON.parse(execFileSync('powershell.exe',['-NoProfile','-Command','Get-Process -Id '+ids.join(',')+' -ErrorAction SilentlyContinue | Select-Object Id,WorkingSet64,PrivateMemorySize64 | ConvertTo-Json -Compress'],{encoding:'utf8'}));
   point.browserProcesses=(Array.isArray(counters)?counters:[counters]).map(p=>({type:processes.find(v=>v.id===p.Id)?.type,pid:p.Id,workingSet:p.WorkingSet64,privateBytes:p.PrivateMemorySize64}));
   point.health=await page.evaluate(()=>({...directHealth,frameAgeMS:performance.now()-directHealth.lastFrame,visibility:document.visibilityState}));expect(point.health.frames).toBeGreaterThan(0);expect(point.health.frameAgeMS).toBeLessThan(5000);expect(point.health.hidden).toBe(false);expect(point.health.contextLost).toBe(false);expect(point.health.visibility).toBe('visible');
   point.audio=await page.evaluate(()=>({voices:window.RiftAudio.voices,context:window.RiftAudio.context?.state}));expect(point.audio.voices).toBe(0);
   point.final=final;point.snapshotTaken=!endpointSnapshots||mission===0||final;
   if(point.snapshotTaken){
   point.snapshot=path.join(root,'heap-'+mission+'.heapsnapshot');const fd=fs.openSync(point.snapshot,'w');const off=browser.cdp.on('HeapProfiler.addHeapSnapshotChunk',event=>fs.writeSync(fd,event.chunk),page.sessionId);
   try{await page.send('HeapProfiler.takeHeapSnapshot',{reportProgress:false},180000);}finally{off();fs.closeSync(fd);}
   point.reachable=summarizeHeap(JSON.parse(fs.readFileSync(point.snapshot,'utf8')));
   }
   report.checkpoints.push(point);save();console.log('Direct inspector='+networkInspection+' mission='+mission+' heap='+point.heap.usedSize+' networkEvents='+point.networkEvents);
  }
  await expedition('warmup');await delay(5000);await checkpoint(0);started=Date.now();let index=0,nextCheckpoint=smoke?durationMS:300000;
  while(Date.now()-started<durationMS||index<(smoke?2:60)){await expedition(++index);if(Date.now()-started>=nextCheckpoint||(smoke&&index===1)||(!smoke&&[49,54,59,60].includes(index))){await checkpoint(index,Date.now()-started>=durationMS&&index>=(smoke?2:60));nextCheckpoint=Date.now()-started+300000;}}
  if(report.checkpoints.at(-1).mission!==index)await checkpoint(index,true);
  expect(report.checkpoints.at(-1).final).toBe(true);
  if(endpointSnapshots){expect(report.checkpoints.filter(p=>p.snapshotTaken).length).toBe(2);expect(report.checkpoints.some(p=>!p.snapshotTaken)).toBe(true);}
  else expect(report.checkpoints.every(p=>p.snapshotTaken)).toBe(true);
  report.measuredDurationMS=Date.now()-started;report.growthBytes=report.checkpoints.at(-1).heap.usedSize-report.checkpoints[0].heap.usedSize;
  if(networkInspection)expect(page.networkEvents).toBeGreaterThan(0);else expect(page.networkEvents).toBe(0);
  report.status='capture_complete_review_required';
 }catch(error){report.status='incomplete';report.errors.push({kind:'capture',message:error.message});throw error;}
 finally{if(page)await page.controls(new Set()).catch(()=>{});await browser.close();report.finishedAt=new Date().toISOString();save();}
});
