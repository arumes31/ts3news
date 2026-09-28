const {test,expect}=require('@playwright/test');
const fs=require('node:fs'),path=require('node:path'),os=require('node:os'),crypto=require('node:crypto');
const {execFileSync}=require('node:child_process');
const git=(...args)=>execFileSync('git',args,{encoding:'utf8',maxBuffer:20*1024*1024}).trim();
for(let sample=1;sample<=3;sample++)test('cached startup sample '+sample,async({page,context,browser},info)=>{
 const network={latency:150,downloadThroughput:200000,uploadThroughput:93750};
 const report={sample,startedAt:new Date().toISOString(),revision:git('rev-parse','HEAD'),trackedDiffSHA256:crypto.createHash('sha256').update(git('diff','HEAD','--binary')).digest('hex'),dirtyFiles:git('status','--short'),
  host:{platform:os.platform(),release:os.release(),cpu:os.cpus()[0]?.model,logicalCPUs:os.cpus().length,totalRAM:os.totalmem(),freeRAM:os.freemem()},browser:browser.version(),
  profile:{viewport:{width:1280,height:900},dpr:1,cpuSlowdown:4,network,headless:true,physicalMinimumDevice:false},
  server:'fresh managed e2e fixture compiled from this checkout; synthetic player only',scenario:'/abyss/rift',thresholdMS:3000,errors:[],gate:'unmeasured',
  command:'node node_modules/@playwright/test/cli.js test --config=playwright.warm-performance.config.js'};
 const output=info.outputPath('warm-start-report.json');fs.mkdirSync(path.dirname(output),{recursive:true});
 fs.writeFileSync(info.outputPath('server-tracked-diff.patch'),git('diff','HEAD','--binary'));
 let measuring=false;
 page.on('pageerror',e=>{if(measuring)report.errors.push({kind:'page',message:e.message});});
 page.on('requestfailed',r=>{if(measuring)report.errors.push({kind:'request',path:new URL(r.url()).pathname,message:r.failure()?.errorText});});
 page.on('response',r=>{if(measuring&&r.status()>=400)report.errors.push({kind:'http',path:new URL(r.url()).pathname,status:r.status()});});
 try{
  const cdp=await context.newCDPSession(page);await cdp.send('Network.enable');await cdp.send('Network.setCacheDisabled',{cacheDisabled:false});
  await cdp.send('Emulation.setCPUThrottlingRate',{rate:4});
  // Each test owns a fresh context. Warm this exact page/build without network
  // throttling, then preserve its HTTP cache for the measured navigation.
  await page.goto(report.scenario);await expect(page.locator('#rift-start')).toBeEnabled({timeout:120000});
  report.build=(await(await page.request.get('/api/abyss/rift')).json()).build;
  await page.locator('.rift-settings > summary').click();await page.locator('#rift-display-preset').selectOption('lowPower');await page.locator('#rift-apply-preset').click();
  report.settings=await page.evaluate(()=>JSON.parse(localStorage.getItem('riftDisplay')));
  expect(report.settings.fps).toBe(30);
  await page.waitForLoadState('networkidle');
  report.warmedArt=await page.evaluate(()=>performance.getEntriesByType('resource').filter(e=>new URL(e.name).pathname.endsWith('.png')).map(e=>new URL(e.name).pathname));
  expect(report.warmedArt.length).toBeGreaterThanOrEqual(13);
  await page.goto('about:blank');
  await page.addInitScript(()=>{
   performance.setResourceTimingBufferSize(2000);
   window.warmStartup={readyMS:null,hidden:document.visibilityState!=='visible'};
   document.addEventListener('visibilitychange',()=>{if(warmStartup.readyMS===null&&document.visibilityState!=='visible')warmStartup.hidden=true;});
   const observer=new MutationObserver(()=>{
    const button=document.getElementById('rift-start');
    if(button&&!button.disabled&&window.RiftRenderer?.atlasProgress.ready&&button.dataset.retry!=='true'){
     warmStartup.readyMS=performance.now();observer.disconnect();
    }
   });observer.observe(document,{subtree:true,childList:true,attributes:true});
  });
  await cdp.send('Network.emulateNetworkConditions',{offline:false,...network,connectionType:'cellular4g'});
  measuring=true;
  await page.goto(report.scenario,{waitUntil:'commit',timeout:30000});
  await page.waitForFunction(()=>warmStartup.readyMS!==null||document.getElementById('rift-start')?.dataset.retry==='true',null,{timeout:120000});
  report.capture=await page.evaluate(()=>({
   ...warmStartup,atlas:RiftRenderer.atlasProgress,button:document.getElementById('rift-start').textContent,
   navigation:performance.getEntriesByType('navigation')[0].toJSON(),
   resources:performance.getEntriesByType('resource').filter(e=>e.responseEnd<=warmStartup.readyMS).map(e=>({path:new URL(e.name).pathname,transferBytes:e.transferSize,encodedBodyBytes:e.encodedBodySize,durationMS:e.duration})),
  }));
  const c=report.capture;expect(c.readyMS).not.toBeNull();expect(c.hidden).toBe(false);expect(c.atlas.ready).toBe(true);
  const art=c.resources.filter(e=>e.path.endsWith('.png'));
  expect(art.length).toBeGreaterThanOrEqual(13);
  report.uncachedArt=art.filter(e=>e.transferBytes!==0||e.encodedBodyBytes===0);
  expect(report.uncachedArt).toEqual([]);
  report.transferBytes=c.navigation.transferSize+c.resources.reduce((sum,r)=>sum+r.transferBytes,0);
  expect((await(await page.request.get('/api/abyss/rift')).json()).build).toEqual(report.build);
  report.gate=report.errors.length||c.readyMS>report.thresholdMS?'fail':'development profile pass only';
  report.scope='Three independent cache-warmed contexts, one navigation each; cache retained across about:blank, no request interception. Measures enabled Start, not first fight, physical hardware or cold startup.';
  console.log(JSON.stringify({sample,readyMS:c.readyMS,transferBytes:report.transferBytes,cachedImages:art.length,gate:report.gate,errors:report.errors}));
 }catch(error){report.gate='invalid capture';report.failure=error.message;throw error;}
 finally{fs.writeFileSync(output,JSON.stringify(report,null,2)+'\n');}
});
