const {test,expect}=require('@playwright/test');
const fs=require('node:fs'),path=require('node:path'),os=require('node:os'),crypto=require('node:crypto');
const {execFileSync}=require('node:child_process');
const git=(...args)=>execFileSync('git',args,{encoding:'utf8',maxBuffer:20*1024*1024}).trim();
const smoke=process.env.BRAWL_FRAME_SMOKE==='1';
const profiling=process.env.BRAWL_FRAME_PROFILE==='1';
const percentile=(values,p)=>{const sorted=[...values].sort((a,b)=>a-b);return sorted[Math.ceil(sorted.length*p)-1]??null;};
for(let sample=1;sample<=(smoke||profiling?1:3);sample++)test('crowded boss frame sample '+sample,async({page,context,browser},info)=>{
 const report={sample,startedAt:new Date().toISOString(),smoke,profiling,revision:git('rev-parse','HEAD'),trackedDiffSHA256:crypto.createHash('sha256').update(git('diff','HEAD','--binary')).digest('hex'),dirtyFiles:git('status','--short'),
  host:{platform:os.platform(),release:os.release(),cpu:os.cpus()[0]?.model,logicalCPUs:os.cpus().length,totalRAM:os.totalmem(),freeRAM:os.freemem()},browser:browser.version(),
  profile:{viewport:{width:1280,height:900},dpr:1,cpuSlowdown:4,preset:'lowPower',headless:true,physicalMinimumDevice:false},
  server:'fresh managed go test -tags=e2e fixture; production simulation, in-memory synthetic character',scenario:'/abyss/rift?scenario=visual&seed=boss-frames-v1&level=100&room=2&subclass=bloodblade&riftFrameDebug=1',
  thresholds:{intervalP95:50,intervalP99:100,renderP95:16},errors:[],gate:'unmeasured'};
 const output=info.outputPath('frame-report.json');fs.mkdirSync(path.dirname(output),{recursive:true});
 const save=()=>fs.writeFileSync(output,JSON.stringify(report,null,2)+'\n');
 fs.writeFileSync(info.outputPath('server-tracked-diff.patch'),git('diff','HEAD','--binary'));
 report.command='node node_modules/@playwright/test/cli.js test --config=playwright.boss-performance.config.js';
 const held=new Set();
 async function controls(wanted){for(const k of [...held])if(!wanted.has(k)){await page.keyboard.up(k);held.delete(k);}for(const k of wanted)if(!held.has(k)){await page.keyboard.down(k);held.add(k);}}
 const read=async()=>{const response=await page.request.get('/api/abyss/rift');expect(response.ok()).toBe(true);return(await response.json()).run;};
 page.on('response',r=>{if(r.status()>=400)report.errors.push({kind:'http',path:new URL(r.url()).pathname,status:r.status()});});
 page.on('pageerror',e=>report.errors.push({kind:'page',message:e.message}));
 page.on('console',m=>{if(m.type()==='error')report.errors.push({kind:'console',message:m.text()});});
 try{
  const browserCDP=await browser.newBrowserCDPSession();const system=await browserCDP.send('SystemInfo.getInfo');report.graphics={devices:system.gpu.devices,featureStatus:system.gpu.featureStatus};await browserCDP.detach();
  const cdp=await context.newCDPSession(page);await cdp.send('Emulation.setCPUThrottlingRate',{rate:4});
  await page.goto(report.scenario);await expect(page.locator('#rift-start')).toBeEnabled({timeout:120000});
  await page.locator('#rift-auto').uncheck();await page.locator('.rift-settings > summary').click();await page.locator('#rift-display-preset').selectOption('lowPower');await page.locator('#rift-apply-preset').click();await page.locator('.rift-settings > summary').click();
  await page.evaluate(()=>RiftRenderer.ready);await page.waitForTimeout(5000);
  const initial=await read();expect(initial.level.id).toBe(100);expect(initial.room).toBe(2);expect(initial.enemies.some(e=>e.kind==='boss')).toBe(true);
  report.initial={enemyCount:initial.enemies.length,bosses:initial.enemies.filter(e=>e.kind==='boss').map(e=>e.name),class:initial.build.class,replaySeed:initial.replay_seed};
  report.settings=await page.evaluate(()=>({display:JSON.parse(localStorage.getItem('riftDisplay')),visibility:document.visibilityState}));expect(report.settings.display.fps).toBe(30);
  await page.evaluate(()=>{
   const r=RiftRenderer,d=r.frameDiagnostics,originalSnapshot=r.snapshot,originalPush=d.samples.push;
   const capture=window.bossFrameCapture={samples:[],started:null,ended:null,status:null,enemyPeak:0,projectilePeak:0,snapshots:0,hidden:false,contextLost:false,active:false};
   document.querySelector('#rift-canvas').addEventListener('contextlost',()=>capture.contextLost=true);
   document.addEventListener('visibilitychange',()=>{if(capture.active&&document.visibilityState!=='visible')capture.hidden=true;});
   r.snapshot=function(run,...args){
    if(run?.status==='fighting'&&!run.paused&&!capture.started){capture.started=performance.now();capture.active=true;d.last=null;}
    if(capture.active){capture.snapshots++;capture.enemyPeak=Math.max(capture.enemyPeak,run.enemies.filter(e=>e.hp>0).length);capture.projectilePeak=Math.max(capture.projectilePeak,run.projectiles.length);if(run.status!=='fighting'||run.paused){capture.ended=performance.now();capture.status=run.status;capture.active=false;}}
    return originalSnapshot.call(this,run,...args);
   };
   d.samples.push=function(sample){if(capture.active)capture.samples.push({...sample,at:performance.now()});return originalPush.call(this,sample);};
  });
  if(profiling){await cdp.send('Profiler.enable');await cdp.send('Profiler.start');}
  await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();
  const deadline=Date.now()+(smoke?5000:60000);let run=initial;
  while(Date.now()<deadline){
   run=await read();if(run.status!=='fighting')break;
   const p=run.player,target=run.enemies.filter(e=>e.hp>0).sort((a,b)=>Math.hypot(a.x-p.x,a.y-p.y)-Math.hypot(b.x-p.x,b.y-p.y))[0],wanted=new Set(['Space']);
   if(target){const dx=target.x-p.x,dy=target.y-p.y;if(Math.abs(dy)>10)wanted.add(dy>0?'s':'w');if(Math.abs(dx)>60||Math.sign(dx)!==p.facing)wanted.add(dx>0?'d':'a');if(Math.abs(dx)<120&&Math.abs(dy)<30){const [builder,finisher]=run.build.signatures;if(run.resource>0&&finisher&&!(run.skill_timers[finisher.id]>0)&&p.mana>=finisher.cost)wanted.add('e');else if(builder&&!(run.skill_timers[builder.id]>0)&&p.mana>=builder.cost)wanted.add('q');else wanted.add('j');}}
   await controls(wanted);await page.waitForTimeout(120);
  }
  await controls(new Set());
  report.capture=await page.evaluate(()=>{const c=bossFrameCapture;c.ended??=performance.now();c.active=false;return c;});
  if(profiling){const {profile}=await cdp.send('Profiler.stop');fs.writeFileSync(info.outputPath('boss.cpuprofile'),JSON.stringify(profile));report.cpuProfile='boss.cpuprofile';}
  report.final={status:run.status,clock:run.clock,playerHP:run.player.hp};
  const c=report.capture;report.durationMS=c.ended-c.started;report.termination=c.status||'sampling deadline';report.summary={frames:c.samples.length,intervalP95:percentile(c.samples.map(s=>s.interval),.95),intervalP99:percentile(c.samples.map(s=>s.interval),.99),renderP95:percentile(c.samples.map(s=>s.render),.95)};
  expect(c.started).not.toBeNull();expect(c.samples.length).toBeGreaterThan(0);expect(c.hidden).toBe(false);expect(c.contextLost).toBe(false);
  const s=report.summary;report.gate=profiling?'unmeasured (profiling instrumentation)':smoke?'unmeasured (smoke only)':report.errors.length||s.intervalP95>50||s.intervalP99>100||s.renderP95>16?'fail':'development profile pass only';
  report.scope='Completed RAF intervals and synchronous canvas submission, not GPU presentation or physical target hardware. Fixture-selected final tier; no artificial crowd or health inflation.';
  console.log(JSON.stringify({sample,durationMS:report.durationMS,final:report.final,peaks:{enemies:c.enemyPeak,projectiles:c.projectilePeak},...s,gate:report.gate}));
 }catch(error){report.gate='invalid capture';report.failure=error.message;throw error;}finally{await controls(new Set());save();}
});
