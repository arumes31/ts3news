const {test,expect}=require('@playwright/test');
const fs=require('node:fs'),path=require('node:path');
test('history reload blocks mutations until fresh state is confirmed',async({page})=>{
 await page.route('**/static/rift.js*',route=>route.fulfill({contentType:'application/javascript',body:fs.readFileSync(path.resolve(__dirname,'../../internal/bot/webassets/rift.js'),'utf8')}));
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
 await page.evaluate(()=>{
  const fetch=window.fetch;window.racePosts=0;window.raceReads=0;
  window.fetch=function(url,options){
   if(String(url).includes('/api/abyss/rift')){
    if(options?.method==='POST')window.racePosts++;
    if(options?.method==='GET'){window.raceReads++;return new Promise((resolve,reject)=>{window.releaseRaceRead=()=>fetch.call(this,url,options).then(resolve,reject);options.signal.addEventListener('abort',()=>reject(new DOMException('aborted','AbortError')),{once:true});});}
   }
   return fetch.call(this,url,options);
  };
  window.dispatchEvent(new PageTransitionEvent('pageshow',{persisted:true}));
  for(let i=0;i<20;i++)document.querySelector('#rift-start').click();
 });
 await page.waitForTimeout(250);
 expect(await page.evaluate(()=>window.raceReads)).toBe(1);
 expect(await page.evaluate(()=>window.racePosts)).toBe(0);
 await expect(page.locator('#rift-start')).toBeDisabled();
 await page.evaluate(()=>window.releaseRaceRead());
 await expect(page.locator('#rift-start')).toBeEnabled();
 await expect(page.locator('#rift-start')).toHaveText('Resume expedition');
});
