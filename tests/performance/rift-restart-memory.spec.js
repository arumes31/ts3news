const {test,expect}=require('@playwright/test');
const fs=require('node:fs');
const os=require('node:os');
const crypto=require('node:crypto');
const {execFileSync}=require('node:child_process');
const smoke=process.env.BRAWL_MEMORY_SMOKE==='1';
const cycles=smoke?1:20;
const git=(...args)=>execFileSync('git',args,{encoding:'utf8',maxBuffer:20*1024*1024}).trim();

// Counts are reachable nodes after GC, not dominator retained sizes. Preserve the
// original snapshot for retaining-path inspection; aggregate counts cannot prove
// which owner retains an object or whether a particular increase is a leak.
function summarizeHeap(file){
 const heap=JSON.parse(fs.readFileSync(file,'utf8'));
 const fields=heap.snapshot.meta.node_fields,n=fields.length;
 const typeIndex=fields.indexOf('type'),nameIndex=fields.indexOf('name');
 const types=heap.snapshot.meta.node_types[typeIndex],byType={},objects={};
 const detachedIndex=fields.indexOf('detachedness');
 let detached=0;
 for(let i=0;i<heap.nodes.length;i+=n){
  const type=types[heap.nodes[i+typeIndex]],name=heap.strings[heap.nodes[i+nameIndex]];
  byType[type]=(byType[type]||0)+1;
  if(type==='object'||type==='native'||type==='closure')objects[name]=(objects[name]||0)+1;
  if(detachedIndex>=0&&heap.nodes[i+detachedIndex]===2)detached++;
 }
 return {nodes:heap.nodes.length/n,byType,detached,objects};
}

for(let sample=1;sample<=(smoke?1:3);sample++)test(`restart memory sample ${sample}`,async({page,context,browser},info)=>{
 const report={startedAt:new Date().toISOString(),mode:smoke?'smoke (not a gate run)':'20 in-page start/fight/exit cycles',sample,cycles,
  server:{command:'go test -tags=e2e ./internal/bot -run TestAbyssE2EServer -count=1 -v -timeout=130m',managedFresh:true,revision:git('rev-parse','HEAD'),trackedDiffSHA256:crypto.createHash('sha256').update(git('diff','HEAD','--binary')).digest('hex'),dirtyFiles:git('status','--short')},
  host:{platform:os.platform(),release:os.release(),cpu:os.cpus()[0]?.model,logicalCPUs:os.cpus().length,totalRAM:os.totalmem(),freeRAM:os.freemem()},
  browser:browser.version(),profile:{viewport:{width:1280,height:900},dpr:1,cpuSlowdown:4,display:'lowPower',physicalMinimumDevice:false},
  scenario:'/abyss/rift?subclass=bloodblade',seed:'production random; replay seeds and run IDs captured per cycle',mission:1,
  protocol:'After each exit: settle 2s, collectGarbage, settle 1s, collectGarbage, getHeapUsage and DOM counters. Snapshots at baseline and every 5 cycles.',
  checkpoints:[],expeditions:[],errors:[],gate:'unmeasured',processMemory:'not collected; JS heap is not browser process memory'};
 const output=info.outputPath('memory-report.json');fs.mkdirSync(require('node:path').dirname(output),{recursive:true});
 const save=()=>fs.writeFileSync(output,JSON.stringify(report,null,2)+'\n');
 fs.writeFileSync(info.outputPath('server-tracked-diff.patch'),git('diff','HEAD','--binary'));
 report.command='node node_modules/@playwright/test/cli.js test --config=playwright.memory.config.js';
 const session=await context.newCDPSession(page);
 const held=new Set();
 async function controls(wanted){
  for(const key of [...held])if(!wanted.has(key)){await page.keyboard.up(key);held.delete(key);}
  for(const key of wanted)if(!held.has(key)){await page.keyboard.down(key);held.add(key);}
 }
 const read=async()=>{const response=await page.request.get('/api/abyss/rift');expect(response.ok()).toBe(true);return(await response.json()).run;};
 async function expedition(index){
  const started=Date.now();await page.locator('#rift-start').click();
  await expect(page.locator('#rift-overlay')).toBeHidden();
  let run;
  try{
   while(Date.now()-started<120000){
    run=await read();expect(run.status,'combat must reach a checkpoint without death').not.toBe('dead');
    if(run.status==='cleared')break;
    expect(run.status).toBe('fighting');
    const p=run.player,target=run.enemies.filter(e=>e.hp>0).sort((a,b)=>Math.abs(a.x-p.x)-Math.abs(b.x-p.x))[0];
    const wanted=new Set(['Space']);
    if(target){
     const dx=target.x-p.x,dy=target.y-p.y;
     if(Math.abs(dy)>10)wanted.add(dy>0?'s':'w');
     if(Math.abs(dx)>60||Math.sign(dx)!==p.facing)wanted.add(dx>0?'d':'a');
     if(Math.abs(dx)<120&&Math.abs(dy)<30){
      const [builder,finisher]=run.build.signatures;
      if(run.resource>0&&!(run.skill_timers[finisher.id]>0)&&p.mana>=finisher.cost)wanted.add('e');
      else if(!(run.skill_timers[builder.id]>0)&&p.mana>=builder.cost)wanted.add('q');
      else wanted.add('j');
     }
    }
    await controls(wanted);await page.waitForTimeout(120);
   }
  }finally{await controls(new Set());}
  expect(run?.status).toBe('cleared');
  await expect(page.locator('#rift-exit')).toBeVisible();await page.locator('#rift-exit').click();
  await expect(page.locator('#rift-overlay')).toBeVisible();
  const ended=await read();expect(ended.id).toBe(run.id);expect(ended.status).not.toBe('fighting');expect(ended.status).not.toBe('cleared');
  report.expeditions.push({index,id:run.id,replaySeed:run.replay_seed,room:run.room,status:ended.status,durationMS:Date.now()-started});save();
 }
 async function checkpoint(index,snapshot){
  await page.waitForTimeout(2000);await session.send('HeapProfiler.collectGarbage');
  await page.waitForTimeout(1000);await session.send('HeapProfiler.collectGarbage');
  const point={cycle:index,at:new Date().toISOString(),heap:await session.send('Runtime.getHeapUsage'),dom:await session.send('Memory.getDOMCounters')};
  if(snapshot){
   point.snapshot=info.outputPath(`heap-${index}.heapsnapshot`);
   const fd=fs.openSync(point.snapshot,'w');const chunk=event=>fs.writeSync(fd,event.chunk);
   session.on('HeapProfiler.addHeapSnapshotChunk',chunk);
   try{await session.send('HeapProfiler.takeHeapSnapshot',{reportProgress:false});}finally{session.off('HeapProfiler.addHeapSnapshotChunk',chunk);fs.closeSync(fd);}
   point.reachable=summarizeHeap(point.snapshot);
  }
  report.checkpoints.push(point);save();console.log(`Memory sample ${sample}, cycle ${index}: ${(point.heap.usedSize/1048576).toFixed(2)} MiB`);
 }
 page.on('console',message=>{if(message.type()==='error')report.errors.push({kind:'console',message:message.text()});});
 page.on('pageerror',error=>report.errors.push({kind:'page',message:error.message}));
 page.on('response',response=>{if(response.status()>=400)report.errors.push({kind:'http',path:new URL(response.url()).pathname,status:response.status()});});
 try{
  await session.send('Emulation.setCPUThrottlingRate',{rate:4});
  await page.goto(report.scenario);await expect(page.locator('#rift-start')).toBeEnabled({timeout:120000});
  await page.locator('#rift-auto').uncheck();
  await page.locator('.rift-settings > summary').click();
  await page.locator('#rift-display-preset').selectOption('lowPower');
  await page.locator('#rift-apply-preset').click();
  await page.locator('.rift-settings > summary').click();
  report.settings=await page.evaluate(()=>({display:JSON.parse(localStorage.getItem('riftDisplay')),audio:localStorage.getItem('riftAudio'),visibility:document.visibilityState}));
  expect(report.settings.display.fps).toBe(30);expect(report.settings.display.particles).toBe(false);
  // Warm an entire real encounter and exit before taking the idle baseline.
  await expedition('warmup');await page.waitForTimeout(5000);await checkpoint(0,true);
  for(let index=1;index<=cycles;index++){await expedition(index);await checkpoint(index,index%5===0||smoke);}
  expect(await page.evaluate(()=>document.visibilityState)).toBe('visible');
  expect(new Set(report.expeditions.map(e=>e.id)).size).toBe(cycles+1);
  const first=report.checkpoints[0],last=report.checkpoints.at(-1);
  report.growthBytes=last.heap.usedSize-first.heap.usedSize;
  report.heapSizeGate=report.growthBytes<=10*1048576?'pass':'fail';
  const final=report.checkpoints.slice(-5);
  report.finalFive=final.map(p=>({cycle:p.cycle,usedSize:p.heap.usedSize,...p.dom}));
  report.gate=smoke?'unmeasured (smoke only)':report.heapSizeGate==='fail'||report.errors.length?'fail':'retaining-path review required';
  expect(report.errors).toEqual([]);expect(report.growthBytes).toBeLessThanOrEqual(10*1048576);
 }catch(error){report.error=error.message;if(report.gate!=='fail')report.gate='incomplete';throw error;}
 finally{report.finishedAt=new Date().toISOString();save();await session.detach();}
});
