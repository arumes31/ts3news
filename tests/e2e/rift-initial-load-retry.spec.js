const {test,expect}=require('@playwright/test');
const fs=require('node:fs'),path=require('node:path');
for(const mode of ['fetch','body','persistent'])test('initial aborted read retries once after artwork: '+mode,async({page})=>{
 await page.route('**/static/rift.js*',route=>route.fulfill({contentType:'application/javascript',body:fs.readFileSync(path.resolve(__dirname,'../../internal/bot/webassets/rift.js'),'utf8')}));
 let release;const gate=new Promise(resolve=>release=resolve);
 await page.route('**/static/rift_regions.png*',async route=>{await gate;await route.continue();});
 await page.addInitScript(mode=>{
  const original=window.fetch;window.initialReads=0;
  window.fetch=function(input,options){
   if(String(input).includes('/api/abyss/rift')&&options?.method==='GET'){
    window.initialReads++;
    if(mode==='persistent'||window.initialReads===1){const error=new DOMException('Fixture initial read timed out','AbortError');return mode==='body'?Promise.resolve({status:200,ok:true,json:()=>Promise.reject(error)}):Promise.reject(error);}
   }
   return original.apply(this,arguments);
  };
 },mode);
 try{
  await page.goto('/abyss/rift',{waitUntil:'domcontentloaded'});
  await expect.poll(()=>page.evaluate(()=>window.initialReads)).toBe(1);
  await expect(page.locator('#rift-start')).toBeDisabled();
 }finally{release();}
 await expect(page.locator('#rift-start')).toBeEnabled({timeout:15000});
 expect(await page.evaluate(()=>window.initialReads)).toBe(2);
 if(mode==='persistent')await expect(page.locator('#rift-start')).toHaveText('Retry loading');
 else await expect(page.locator('#rift-start')).toHaveText('Enter the ruins →');
});
