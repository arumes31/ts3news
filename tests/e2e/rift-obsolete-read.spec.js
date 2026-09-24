const {test,expect}=require('@playwright/test');
const fs=require('node:fs'),path=require('node:path');
test('page exit cancels pending read without retry and history return reads fresh state',async({page})=>{
 await page.route('**/static/rift.js*',route=>route.fulfill({contentType:'application/javascript',body:fs.readFileSync(path.resolve(__dirname,'../../internal/bot/webassets/rift.js'),'utf8')}));
 await page.addInitScript(()=>{
  const fetch=window.fetch;window.readAttempts=0;window.readAborts=0;
  window.fetch=function(url,options){
   if(String(url).includes('/api/abyss/rift')&&options?.method==='GET'){
    window.readAttempts++;
    if(window.readAttempts===1)return new Promise((resolve,reject)=>{options.signal.addEventListener('abort',()=>{window.readAborts++;reject(new DOMException('Navigation cancelled read','AbortError'));},{once:true});});
   }
   return fetch.call(this,url,options);
  };
 });
 await page.goto('/abyss/rift?scenario=checkpoint');
 await page.waitForFunction(()=>window.readAttempts===1);
 await page.evaluate(()=>window.dispatchEvent(new PageTransitionEvent('pagehide',{persisted:true})));
 await expect.poll(()=>page.evaluate(()=>window.readAborts),{timeout:1500}).toBe(1);
 await page.evaluate(()=>window.RiftRenderer.ready);
 await page.waitForTimeout(150);
 expect(await page.evaluate(()=>window.readAttempts)).toBe(1);
 await expect(page.locator('#rift-start')).not.toHaveText('Retry loading');
 await page.evaluate(()=>window.dispatchEvent(new PageTransitionEvent('pageshow',{persisted:true})));
 await expect(page.locator('#rift-start')).toBeEnabled();
 expect(await page.evaluate(()=>window.readAttempts)).toBe(2);
 await expect(page.locator('#rift-start')).toHaveText('Resume expedition');
});
