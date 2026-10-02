const {test,expect,chromium}=require('@playwright/test');
const fs=require('node:fs/promises');
const path=require('node:path');

// Use Chromium's actual tab zoom; viewport resizing alone is not zoom evidence.
test('native browser zoom keeps Brawl controls reachable from 125 through 400 percent',async({baseURL},testInfo)=>{
 test.setTimeout(180000);
 const extension=testInfo.outputPath('zoom-extension');await fs.mkdir(extension,{recursive:true});
 await fs.writeFile(path.join(extension,'manifest.json'),JSON.stringify({manifest_version:3,name:'Brawl isolated zoom test',version:'1.0',permissions:['tabs'],background:{service_worker:'background.js'}}));
 await fs.writeFile(path.join(extension,'background.js'),'chrome.runtime.onInstalled.addListener(()=>{});');
 const context=await chromium.launchPersistentContext('',{channel:'chromium',headless:true,viewport:{width:1280,height:900},args:['--disable-extensions-except='+extension,'--load-extension='+extension]});
 const results=[];
 try{
  const worker=context.serviceWorkers()[0]||await context.waitForEvent('serviceworker');
  const page=context.pages()[0]||await context.newPage();await page.goto(baseURL+'/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
  const baseline=await page.evaluate(()=>({width:innerWidth,ratio:devicePixelRatio}));
  const tabID=await worker.evaluate(async url=>(await chrome.tabs.query({})).find(tab=>tab.url===url).id,page.url());
  async function capture(name){const cdp=await context.newCDPSession(page);try{const shot=await cdp.send('Page.captureScreenshot',{format:'png',captureBeyondViewport:false});await fs.writeFile(testInfo.outputPath(name),Buffer.from(shot.data,'base64'));}finally{await cdp.detach();}}
  async function reachable(selector){
   const control=page.locator(selector);await expect(control).toBeVisible();await control.evaluate(node=>node.scrollIntoView({block:'center',inline:'nearest',behavior:'instant'}));
   const state=await control.evaluate(node=>{const b=node.getBoundingClientRect(),hit=document.elementFromPoint(b.x+b.width/2,b.y+b.height/2);return {fits:b.left>=-1&&b.right<=innerWidth+1&&b.top>=-1&&b.bottom<=innerHeight+1,hit:!!hit&&(hit===node||node.contains(hit))};});
   if(!state.fits||!state.hit){await capture('obstructed-control.png');const obstruction=await control.evaluate(node=>{const b=node.getBoundingClientRect(),hit=document.elementFromPoint(b.x+b.width/2,b.y+b.height/2);return {bounds:b.toJSON(),hit:hit?.outerHTML.slice(0,800),width:innerWidth,height:innerHeight};});expect(state,selector+' '+JSON.stringify(obstruction)).toEqual({fits:true,hit:true});}
  }
  for(const factor of [1.25,1.5,2,3,4]){
   await worker.evaluate(async({tabID,factor})=>chrome.tabs.setZoom(tabID,factor),{tabID,factor});
   expect(await worker.evaluate(id=>chrome.tabs.getZoom(id),tabID)).toBeCloseTo(factor,8);
   await expect.poll(()=>page.evaluate(()=>devicePixelRatio)).toBeCloseTo(baseline.ratio*factor,2);
   const metrics=await page.evaluate(()=>({width:innerWidth,height:innerHeight,ratio:devicePixelRatio}));
   expect(metrics.width).toBe(Math.floor(baseline.width/factor));
   await page.goto(baseURL+'/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
   await reachable('#rift-start');await page.locator('#rift-auto').uncheck();
   await reachable('.rift-settings > summary');await page.locator('.rift-settings > summary').click();await page.locator('#rift-text-scale').selectOption('1.25');
   expect(await page.locator('#rift-app').evaluate(node=>getComputedStyle(node).getPropertyValue('--rift-hud-scale').trim())).toBe('1.25');
   await reachable('#rift-prepare-diagnostics');await page.locator('.rift-settings > summary').click();
   await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();
   for(const selector of ['#rift-controls-open','#rift-pause','#rift-next','#rift-exit','.rift-basics button[data-bind="attack"]','.rift-basics button[data-bind="jump"]','.rift-basics button[data-bind="guard"]'])await reachable(selector);
   for(const selector of ['.rift-stage-top button','.rift-basics button','.rift-skills button','.rift-signatures button'])for(let i=0;i<await page.locator(selector).count();i++)await reachable(selector+':nth-child('+(i+1)+')');
   await page.locator('#rift-controls-open').click();await expect(page.locator('#rift-controls-dialog')).toBeVisible();await reachable('#rift-controls-close');await page.locator('#rift-controls-close').click();
   await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();await page.locator('#rift-exit').click();await expect(page.locator('#rift-banked')).toHaveText('30 gold · 1 item');
   await reachable('#rift-start');expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth+1)).toBe(true);
   await capture('native-zoom-'+Math.round(factor*100)+'.png');results.push({factor,...metrics});
  }
  await testInfo.attach('native-zoom-measurements',{body:JSON.stringify(results,null,2),contentType:'application/json'});
 }finally{await context.close();}
});
