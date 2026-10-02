const {test,expect}=require('@playwright/test');
test('campaign startup does not fetch unused legacy backgrounds',async({page})=>{
 const requests=[];page.on('request',r=>{if(/rift_(?:boss_)?area\.png/.test(r.url()))requests.push(r.url());});
 await page.goto('/abyss/rift?scenario=visual');await expect(page.locator('#rift-start')).toBeEnabled();
 expect(requests).toEqual([]);
});
for(const room of [0,2])test('saved legacy scene '+room+' waits for its art and renders it',async({page})=>{
 const asset=room===2?'rift_boss_area.png':'rift_area.png';
 await page.addInitScript(asset=>{window.legacyDraws=0;const draw=CanvasRenderingContext2D.prototype.drawImage;CanvasRenderingContext2D.prototype.drawImage=function(image,...args){if(image.src?.includes(asset))window.legacyDraws++;return draw.call(this,image,...args);};},asset);
 let release;const held=new Promise(resolve=>release=resolve);let requested=0;
 await page.route('**/static/'+asset+'*',async route=>{requested++;await held;await route.continue();});
 await page.route('**/api/abyss/rift',async route=>{const response=await route.fetch();const data=await response.json();delete data.run.level;data.run.room=room;await route.fulfill({response,json:data});});
 await page.goto('/abyss/rift?scenario=visual',{waitUntil:'domcontentloaded'});
 await expect.poll(()=>requested).toBe(1);await expect(page.locator('#rift-start')).toBeDisabled();
 release();await expect(page.locator('#rift-start')).toBeEnabled();
 await expect.poll(()=>page.evaluate(()=>window.legacyDraws)).toBeGreaterThan(0);
 await page.evaluate(async room=>{await RiftRenderer.prepareRun({room});await RiftRenderer.prepareRun({room});},room);
 expect(requested).toBe(1);
});

for(const room of [0,2])test('failed deferred artwork '+room+' retries without a cached rejection',async({page})=>{
 const asset=room===2?'rift_boss_area.png':'rift_area.png';
 await page.goto('/abyss/rift?scenario=visual');await expect(page.locator('#rift-start')).toBeEnabled();
 let attempts=0;
 await page.route('**/static/'+asset+'*',async route=>{attempts++;if(attempts===1)await route.abort();else await route.continue();});
 const failure=await page.evaluate(room=>RiftRenderer.prepareRun({room}).then(()=>'',error=>error.message),room);
 expect(failure).toContain('Could not load saved '+(room===2?'boss':'area')+' scene artwork');
 await page.evaluate(room=>Promise.all([RiftRenderer.prepareRun({room}),RiftRenderer.prepareRun({room})]),room);
 expect(attempts).toBe(2);
});
