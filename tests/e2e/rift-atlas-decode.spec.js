const {test,expect}=require('@playwright/test');
const fs=require('node:fs'),path=require('node:path');
for(const asset of ['rift_region_0.png','abyss_combat_roles_v2.png'])test('Start waits for decoding '+asset,async({page})=>{
 await page.route('**/static/rift_renderer.js*',route=>route.fulfill({contentType:'application/javascript',body:fs.readFileSync(path.resolve(__dirname,'../../internal/bot/webassets/rift_renderer.js'),'utf8')}));
 await page.addInitScript(asset=>{
  const decode=HTMLImageElement.prototype.decode;
  const gate=new Promise(resolve=>window.releaseAtlasDecode=resolve);window.atlasDecodeCalls=[];
  HTMLImageElement.prototype.decode=function(){window.atlasDecodeCalls.push(new URL(this.src).pathname);return this.src.includes(asset)?gate.then(()=>decode.call(this)):decode.call(this);};
 },asset);
 await page.goto('/abyss/rift',{waitUntil:'domcontentloaded'});
 await expect.poll(()=>page.evaluate(asset=>window.atlasDecodeCalls.some(url=>url.endsWith(asset)),asset)).toBe(true);
 await expect(page.locator('#rift-start')).toBeDisabled();
 await page.evaluate(()=>window.releaseAtlasDecode());
 await expect(page.locator('#rift-start')).toBeEnabled();
 const decoded=await page.evaluate(()=>window.atlasDecodeCalls);
 expect(new Set(decoded).size).toBeGreaterThanOrEqual(12);
 await expect(page.locator('#rift-atlas-progress')).toHaveText('Critical atlases loaded (7/7)');
});
for(const asset of ['rift_region_0.png','abyss_combat_roles_v2.png'])test('failed decode offers artwork reload: '+asset,async({page})=>{
 await page.route('**/static/rift_renderer.js*',route=>route.fulfill({contentType:'application/javascript',body:fs.readFileSync(path.resolve(__dirname,'../../internal/bot/webassets/rift_renderer.js'),'utf8')}));
 const errors=[];page.on('pageerror',error=>errors.push(error.message));
 await page.addInitScript(asset=>{const decode=HTMLImageElement.prototype.decode;HTMLImageElement.prototype.decode=function(){return this.src.includes(asset)?Promise.reject(new Error('Fixture decode failure')):decode.call(this);};},asset);
 await page.goto('/abyss/rift');
 await expect(page.locator('#rift-start')).toHaveText('Reload artwork');
 await expect(page.locator('#rift-start')).toHaveAttribute('data-artwork-retry','true');
 if(asset==='rift_region_0.png')expect(await page.evaluate(()=>RiftRenderer.atlasProgress.ready)).toBe(false);
 expect(errors).toEqual([]);
});
test('load events remain a compatibility fallback without image decode',async({page})=>{
 await page.route('**/static/rift_renderer.js*',route=>route.fulfill({contentType:'application/javascript',body:fs.readFileSync(path.resolve(__dirname,'../../internal/bot/webassets/rift_renderer.js'),'utf8')}));
 await page.addInitScript(()=>{HTMLImageElement.prototype.decode=undefined;});
 await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toHaveText('Enter the ruins →');await expect(page.locator('#rift-start')).toBeEnabled();
 expect(await page.evaluate(()=>RiftRenderer.atlasProgress.ready)).toBe(true);
});
