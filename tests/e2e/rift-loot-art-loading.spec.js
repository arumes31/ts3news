const {test,expect}=require('@playwright/test');

test('idle preview needs no loot atlas',async({page})=>{
 const paths=[];page.on('request',r=>{if(r.url().includes('/rift_items.png'))paths.push(r.url());});
 await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();
 expect(paths).toEqual([]);
});

for(const saved of [false,true])test((saved?'saved loot':'new encounter')+' waits for loot decoding before play',async({page})=>{
 await page.addInitScript(()=>{
  const decode=HTMLImageElement.prototype.decode;window.lootDecodeCount=0;
  const gate=new Promise(resolve=>window.releaseLootDecode=resolve);
  HTMLImageElement.prototype.decode=function(){if(!this.src.includes('/rift_items.png'))return decode.call(this);window.lootDecodeCount++;return gate.then(()=>decode.call(this));};
 });
 await page.goto('/abyss/rift'+(saved?'?scenario=checkpoint':''),{waitUntil:'domcontentloaded'});
 if(!saved){await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('#rift-start').click();}
 await expect.poll(()=>page.evaluate(()=>window.lootDecodeCount)).toBe(1);
 await expect(page.locator('#rift-overlay')).toBeVisible();await expect(page.locator('#rift-start')).toBeDisabled();
 await page.evaluate(()=>window.releaseLootDecode());
 if(saved)await expect(page.locator('#rift-start')).toBeEnabled();else await expect(page.locator('#rift-overlay')).toBeHidden();
 expect(await page.evaluate(()=>window.lootDecodeCount)).toBe(1);
});

test('invalid loot dimensions reject and concurrent retry requests share a fresh URL',async({page})=>{
 await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();
 let fail=true;const urls=[];
 await page.route('**/static/rift_items.png*',route=>{urls.push(route.request().url());return fail?route.fulfill({contentType:'image/svg+xml',body:'<svg xmlns="http://www.w3.org/2000/svg" width="1" height="1"/>'}):route.continue();});
 const error=await page.evaluate(async()=>{try{await RiftRenderer.prepareRun({level:{id:1,region:0}});return '';}catch(e){return e.message;}});expect(error).toContain('loot artwork');
 fail=false;await page.evaluate(()=>Promise.all([RiftRenderer.prepareRun({level:{id:1,region:0}}),RiftRenderer.prepareRun({level:{id:1,region:0}})]));
 expect(urls).toHaveLength(2);expect(urls[1]).toMatch(/[?&]retry=1$/);
});
