const {test,expect}=require('@playwright/test');
const fs=require('node:fs'),path=require('node:path');
test('repeated recovery reads wait for pending mutation and coalesce',async({page})=>{
 await page.route('**/static/rift.js*',route=>route.fulfill({contentType:'application/javascript',body:fs.readFileSync(path.resolve(__dirname,'../../internal/bot/webassets/rift.js'),'utf8')}));
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
 await page.evaluate(()=>{
  const fetch=window.fetch;window.queueProbe={active:0,peak:0,reads:0,posts:0};
  window.fetch=async function(url,options){
   if(!String(url).includes('/api/abyss/rift'))return fetch.call(this,url,options);
   const state=window.queueProbe;state.active++;state.peak=Math.max(state.peak,state.active);
   try{
    if(options.method==='POST'){state.posts++;if(state.posts===1)await new Promise(resolve=>window.releaseQueuedMutation=resolve);}
    else state.reads++;
    return await fetch.call(this,url,options);
   }finally{state.active--;}
  };
 });
 await page.locator('#rift-start').click();
 await page.waitForFunction(()=>window.queueProbe.posts===1);
 await page.evaluate(()=>{for(let i=0;i<20;i++)window.dispatchEvent(new PageTransitionEvent('pageshow',{persisted:true}));});
 await page.waitForTimeout(150);
 expect(await page.evaluate(()=>window.queueProbe.reads)).toBe(0);
 await page.evaluate(()=>window.releaseQueuedMutation());
 await expect.poll(()=>page.evaluate(()=>window.queueProbe.reads)).toBe(1);
 await expect(page.locator('#rift-start')).toBeEnabled();
 expect(await page.evaluate(()=>window.queueProbe.peak)).toBe(1);
 await expect(page.locator('#rift-start')).toHaveText('Resume expedition');
});
