const {test,expect}=require('@playwright/test');
const fs=require('node:fs'),path=require('node:path'),os=require('node:os'),crypto=require('node:crypto');
const {execFileSync}=require('node:child_process');
const git=(...args)=>execFileSync('git',args,{encoding:'utf8',maxBuffer:20*1024*1024}).trim();
const {installCanvasCostProbe}=require('../../scripts/brawl-canvas-cost.cjs');
const {startTimeline}=require('../../scripts/brawl-timeline.cjs');
const timeline=process.env.BRAWL_FRAME_TRACE==='1';
const opaque=process.env.BRAWL_OPAQUE_CANVAS_EXPERIMENT==='1';
const sharedOrigin=process.env.BRAWL_SHARED_ORIGIN_EXPERIMENT==='1';
if(sharedOrigin&&(opaque||process.env.BRAWL_PROP_BITMAP_EXPERIMENT==='1'||process.env.BRAWL_RENDER_ABLATION))throw Error('Choose one rendering experiment');
if(opaque&&(process.env.BRAWL_PROP_BITMAP_EXPERIMENT==='1'||process.env.BRAWL_RENDER_ABLATION))throw Error('Choose one rendering experiment');
const ablation=process.env.BRAWL_RENDER_ABLATION||'';
if(ablation&&!['actors','background'].includes(ablation))throw Error('Unknown rendering ablation');
if(ablation&&process.env.BRAWL_PROP_BITMAP_EXPERIMENT==='1')throw Error('Choose one rendering experiment');
const canvasCosts=process.env.BRAWL_CANVAS_COST==='1';
const smoke=process.env.BRAWL_FRAME_SMOKE==='1',cpuProfiling=process.env.BRAWL_FRAME_PROFILE==='1'||canvasCosts,profiling=cpuProfiling||timeline||!!ablation;
const percentile=(values,p)=>{const sorted=[...values].sort((a,b)=>a-b);return sorted[Math.ceil(sorted.length*p)-1]??null;};
for(let sample=1;sample<=((smoke||profiling)?1:3);sample++)test('paused crowd120 frame sample '+sample,async({page,context,browser},info)=>{
 const report={sample,startedAt:new Date().toISOString(),smoke,profiling,timelineEnabled:timeline,revision:git('rev-parse','HEAD'),trackedDiffSHA256:crypto.createHash('sha256').update(git('diff','HEAD','--binary')).digest('hex'),dirtyFiles:git('status','--short'),
  host:{platform:os.platform(),release:os.release(),cpu:os.cpus()[0]?.model,logicalCPUs:os.cpus().length,totalRAM:os.totalmem(),freeRAM:os.freemem()},browser:browser.version(),
  profile:{viewport:{width:1280,height:900},dpr:1,cpuSlowdown:4,preset:'lowPower',headless:true,physicalMinimumDevice:false},
  server:'fresh managed e2e fixture; no real player data',scenario:'/abyss/rift?scenario=visual&seed=crowded-v1&crowd=120&riftFrameDebug=1',
  thresholds:{intervalP95:50,intervalP99:100,renderP95:16},errors:[],gate:'unmeasured'};
 const output=info.outputPath('crowd-report.json');fs.mkdirSync(path.dirname(output),{recursive:true});
 const save=()=>fs.writeFileSync(output,JSON.stringify(report,null,2)+'\n');
 fs.writeFileSync(info.outputPath('server-tracked-diff.patch'),git('diff','HEAD','--binary'));
 report.command='node node_modules/@playwright/test/cli.js test --config=playwright.crowd-performance.config.js';
 page.on('pageerror',e=>report.errors.push({kind:'page',message:e.message}));
 page.on('console',m=>{if(m.type()==='error')report.errors.push({kind:'console',message:m.text()});});
 page.on('response',r=>{if(r.status()>=400)report.errors.push({kind:'http',path:new URL(r.url()).pathname,status:r.status()});});
 let timelineCDP,timelineCapture;
 try{
  const browserCDP=await browser.newBrowserCDPSession();const system=await browserCDP.send('SystemInfo.getInfo');report.graphics={devices:system.gpu.devices,featureStatus:system.gpu.featureStatus};await browserCDP.detach();
  const cdp=await context.newCDPSession(page);await cdp.send('Emulation.setCPUThrottlingRate',{rate:4});
  if(process.env.BRAWL_PROP_BITMAP_EXPERIMENT==='1'){
   const source=fs.readFileSync(path.resolve(__dirname,'../../internal/bot/webassets/rift_renderer.js'),'utf8');
   const marker='  // Saved expeditions and current build previews can require different sheets.';
   expect(source.includes(marker)).toBe(true);
   const candidate=source.replace(marker,'  renderer.ready=renderer.ready.then(async()=>{images.props=await createImageBitmap(images.props);});\n'+marker);
   report.experiment={kind:'full-size prop ImageBitmap',rendererSHA256:crypto.createHash('sha256').update(candidate).digest('hex')};
   fs.writeFileSync(info.outputPath('experimental-renderer.js'),candidate);
   await page.route('**/static/rift_renderer.js*',route=>route.fulfill({contentType:'application/javascript',body:candidate}));
  }
  if(sharedOrigin){
   const source=fs.readFileSync(path.resolve(__dirname,'../../internal/bot/webassets/rift_renderer.js'),'utf8');
   const candidate=require('../../scripts/brawl-shared-origin-experiment.cjs').sharedOriginCandidate(source);
   report.experiment={kind:'shared atlas origin prefixes',rendererSHA256:crypto.createHash('sha256').update(candidate).digest('hex')};
   fs.writeFileSync(info.outputPath('experimental-renderer.js'),candidate);
   await page.route('**/static/rift_renderer.js*',route=>route.fulfill({contentType:'application/javascript',body:candidate}));
  }
  if(opaque){
   const source=fs.readFileSync(path.resolve(__dirname,'../../internal/bot/webassets/rift_renderer.js'),'utf8');
   expect(source.split("canvas.getContext('2d')").length-1).toBe(1);
   const candidate=source.replace("canvas.getContext('2d')","canvas.getContext('2d',{alpha:false})");
   report.experiment={kind:'opaque main canvas',rendererSHA256:crypto.createHash('sha256').update(candidate).digest('hex')};
   fs.writeFileSync(info.outputPath('experimental-renderer.js'),candidate);
   await page.route('**/static/rift_renderer.js*',route=>route.fulfill({contentType:'application/javascript',body:candidate}));
  }
  if(ablation){
   const source=fs.readFileSync(path.resolve(__dirname,'../../internal/bot/webassets/rift_renderer.js'),'utf8');
   let candidate;
   if(ablation==='actors'){
    const marker='  function actor(unit, now) {';expect(source.includes(marker)).toBe(true);
    candidate=source.replace(marker,marker+'return; // diagnostic only: omit actors\n');
   }else{
    expect(source.split('drawAtlas(background,').length-1).toBe(2);
    candidate=source.replaceAll('drawAtlas(background,','void(background,');
   }
   report.experiment={kind:'diagnostic omission: '+ablation,rendererSHA256:crypto.createHash('sha256').update(candidate).digest('hex')};
   fs.writeFileSync(info.outputPath('experimental-renderer.js'),candidate);
   await page.route('**/static/rift_renderer.js*',route=>route.fulfill({contentType:'application/javascript',body:candidate}));
  }
  await page.goto(report.scenario);await expect(page.locator('#rift-start')).toBeEnabled({timeout:120000});
  await page.locator('.rift-settings > summary').click();await page.locator('#rift-display-preset').selectOption('lowPower');await page.locator('#rift-apply-preset').click();await page.locator('.rift-settings > summary').click();
  report.canvasAttributes=await page.evaluate(()=>document.getElementById('rift-canvas').getContext('2d').getContextAttributes());expect(report.canvasAttributes.alpha).toBe(!opaque);
  const run=(await(await page.request.get('/api/abyss/rift')).json()).run;expect(run.enemies).toHaveLength(120);expect(run.paused).toBe(true);
  report.scene={count:run.enemies.length,alive:run.enemies.filter(e=>e.hp>0).length,bosses:run.enemies.filter(e=>e.kind==='boss').length,projectiles:run.projectiles.length,class:run.build.class,replaySeed:run.replay_seed,clock:run.clock};expect(report.scene.bosses).toBeGreaterThan(0);
  await page.evaluate(async run=>{await RiftRenderer.ready;RiftRenderer.snapshot(run,true);document.querySelector('#rift-overlay').hidden=true;},run);
  await page.locator('#rift-canvas').scrollIntoViewIfNeeded();await page.waitForTimeout(5000);
  report.settings=await page.evaluate(()=>({display:JSON.parse(localStorage.getItem('riftDisplay')),visibility:document.visibilityState,camera:RiftRenderer.cameraFraming,cache:RiftRenderer.atlasCacheStats()}));expect(report.settings.display.fps).toBe(30);
  await page.evaluate(()=>{
   const d=RiftRenderer.frameDiagnostics,originalPush=d.samples.push;
   const capture=window.crowdFrameCapture={started:performance.now(),active:true,samples:[],hidden:false,contextLost:false};d.last=null;
   document.addEventListener('visibilitychange',()=>{if(capture.active&&document.visibilityState!=='visible')capture.hidden=true;});document.querySelector('#rift-canvas').addEventListener('contextlost',()=>capture.contextLost=true);
   d.samples.push=function(sample){if(capture.active)capture.samples.push({...sample,at:performance.now()});return originalPush.call(this,sample);};
  });
  if(canvasCosts)await page.evaluate(installCanvasCostProbe);
  if(timeline){timelineCDP=await browser.newBrowserCDPSession();timelineCapture=await startTimeline(timelineCDP,info.outputPath('crowd.timeline.json'));}
  if(cpuProfiling){await cdp.send('Profiler.enable');await cdp.send('Profiler.start');}
  for(let chunk=0;chunk<(smoke?1:6);chunk++){await page.waitForTimeout(10000);if(chunk%2===1)console.log('Crowd sample '+sample+': '+(chunk+1)*10+'s collected');}
  report.capture=await page.evaluate(()=>{const c=crowdFrameCapture;c.active=false;c.ended=performance.now();c.camera=RiftRenderer.cameraFraming;c.cache=RiftRenderer.atlasCacheStats();return c;});
  if(canvasCosts)report.canvasCosts=await page.evaluate(()=>brawlCanvasCost.stop());
  if(timelineCapture){report.timeline=await timelineCapture.stop();expect(report.timeline.dataLossOccurred).toBe(false);}
  if(cpuProfiling){const {profile}=await cdp.send('Profiler.stop');fs.writeFileSync(info.outputPath('crowd.cpuprofile'),JSON.stringify(profile));report.cpuProfile='crowd.cpuprofile';}
  const c=report.capture;report.durationMS=c.ended-c.started;report.summary={frames:c.samples.length,intervalP95:percentile(c.samples.map(s=>s.interval),.95),intervalP99:percentile(c.samples.map(s=>s.interval),.99),renderP95:percentile(c.samples.map(s=>s.render),.95)};
  expect(c.samples.length).toBeGreaterThan(0);expect(c.hidden).toBe(false);expect(c.contextLost).toBe(false);expect(c.camera.x).toBe(report.settings.camera.x);expect(c.cache.bytes).toBeLessThanOrEqual(c.cache.limitBytes);
  const after=(await(await page.request.get('/api/abyss/rift')).json()).run;expect(after.paused).toBe(true);expect(after.clock).toBe(run.clock);expect(after.enemies).toHaveLength(120);
  const s=report.summary;report.gate=profiling?'unmeasured (profiling instrumentation)':smoke?'unmeasured (smoke only)':report.errors.length||s.intervalP95>50||s.intervalP99>100||s.renderP95>16?'fail':'development profile pass only';
  report.scope='Paused seeded drawing load with normal camera/culling. No combat inputs, enemy AI or projectile load. Not physical target hardware.';
  console.log(JSON.stringify({sample,durationMS:report.durationMS,...s,cache:c.cache,gate:report.gate}));
 }catch(error){report.gate='invalid capture';report.failure=error.message;throw error;}finally{
  if(timelineCapture&&!report.timeline){try{report.timeline=await timelineCapture.stop();}catch(error){report.timelineFailure=error.message;}}
  if(timelineCDP)await timelineCDP.detach().catch(()=>{});save();
 }
});
