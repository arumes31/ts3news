'use strict';
const {chromium}=require('@playwright/test');
const {execFileSync}=require('node:child_process');
const path=require('node:path');
async function main(){
 const target=new URL(process.argv[2]||'http://127.0.0.1:18096/abyss/rift');
 if(!['http:','https:'].includes(target.protocol)||target.username||target.password)throw new Error('Supply an HTTP(S) Brawl page URL without credentials');
 const network={latency:150,downloadThroughput:200000,uploadThroughput:93750};
 const browser=await chromium.launch();
 const report={checkedOutRevision:execFileSync('git',['rev-parse','HEAD'],{cwd:path.resolve(__dirname,'..'),encoding:'utf8'}).trim(),serverRevision:null,browser:browser.version(),platform:process.platform,readinessTimeoutMS:360000,viewport:{width:1280,height:900},network,samples:[]};
 try{
  for(let index=0;index<3;index++){
   const context=await browser.newContext({viewport:report.viewport,locale:'en-US',timezoneId:'UTC',serviceWorkers:'block'});
   const errors=[];
   try{
    const page=await context.newPage(),session=await context.newCDPSession(page);
    page.on('pageerror',error=>errors.push({kind:'page',message:error.message}));
    page.on('requestfailed',request=>errors.push({kind:'request',path:new URL(request.url()).pathname,method:request.method(),message:request.failure()?.errorText}));
    page.on('response',response=>{if(response.status()>=400)errors.push({kind:'http',path:new URL(response.url()).pathname,status:response.status()});});
    await session.send('Network.enable');
    await session.send('Network.setCacheDisabled',{cacheDisabled:true});
    await session.send('Network.emulateNetworkConditions',{offline:false,...network,connectionType:'cellular4g'});
    await page.addInitScript(()=>{
     performance.setResourceTimingBufferSize(2000);
     window.__brawlColdReady=null;
     const observer=new MutationObserver(()=>{
      const button=document.getElementById('rift-start'),progress=window.RiftRenderer?.atlasProgress;
      if(button&&!button.disabled&&progress?.ready&&button.textContent!=='Retry loading'){
       window.__brawlColdReady=performance.now();observer.disconnect();
      }
     });
     observer.observe(document,{subtree:true,childList:true,attributes:true});
    });
    await page.goto(target.href,{waitUntil:'commit',timeout:30000});
    await page.waitForFunction(()=>window.__brawlColdReady!==null||document.getElementById('rift-start')?.dataset.retry==='true',{},{timeout:report.readinessTimeoutMS});
    if(await page.evaluate(()=>window.__brawlColdReady===null))throw new Error('Initial loading failed before readiness');
    const sample=await page.evaluate(()=>{
     const navigation=performance.getEntriesByType('navigation')[0];
     const resources=performance.getEntriesByType('resource').filter(entry=>entry.responseEnd<=window.__brawlColdReady);
     const all=[navigation,...resources];
     return {readyMS:window.__brawlColdReady,domContentLoadedMS:navigation.domContentLoadedEventEnd,transferBytes:all.reduce((sum,e)=>sum+e.transferSize,0),encodedBodyBytes:all.reduce((sum,e)=>sum+e.encodedBodySize,0),resourceCount:resources.length,atlas:window.RiftRenderer.atlasProgress,largest:resources.map(e=>({path:new URL(e.name).pathname,bytes:e.encodedBodySize,durationMS:e.duration})).sort((a,b)=>b.bytes-a.bytes).slice(0,10)};
    });
    sample.errors=errors;
    sample.recoveredInitialReadTimeout=errors.length===1&&errors[0].kind==='request'&&errors[0].method==='GET'&&errors[0].path==='/api/abyss/rift'&&errors[0].message==='net::ERR_ABORTED';
    report.samples.push(sample);
    process.stderr.write(`Cold sample ${index+1}: ${(sample.readyMS/1000).toFixed(2)}s, ${sample.transferBytes} transferred bytes\n`);
    if(errors.length&&!sample.recoveredInitialReadTimeout)throw new Error('Unexpected browser/request errors during cold load');
   }catch(error){
    report.error=error.message;report.failure={errors};
    try{report.failure.snapshot=await context.pages()[0].evaluate(()=>({elapsedMS:performance.now(),button:document.getElementById('rift-start')?.textContent,atlas:window.RiftRenderer?.atlasProgress,completedResources:performance.getEntriesByType('resource').map(e=>({path:new URL(e.name).pathname,bytes:e.encodedBodySize,durationMS:e.duration}))}));}catch(_){}
    process.exitCode=1;break;
   }finally{await context.close();}
  }
  if(report.samples.length===3&&!report.error){const times=report.samples.map(s=>s.readyMS).sort((a,b)=>a-b);report.readinessMS={min:times[0],median:times[1],max:times[2]};}
 }finally{await browser.close();process.stdout.write(JSON.stringify(report,null,2)+'\n');}
}
main().catch(error=>{process.stderr.write(error.message+'\n');process.exitCode=2;});
