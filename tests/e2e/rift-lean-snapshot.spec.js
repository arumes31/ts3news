const {test,expect}=require('@playwright/test');
test('live movement uses lean responses and reload restores full baseline',async({page},info)=>{
 const errors=[],samples=[];page.on('pageerror',e=>errors.push(e.message));
 page.on('response',async response=>{if(new URL(response.url()).pathname!=='/api/abyss/rift'||!response.ok())return;const data=await response.json();samples.push({data,bytes:Buffer.byteLength(JSON.stringify(data)),method:response.request().method()});});
 await page.goto('/abyss/rift?scenario=spawn-hazards');await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('#rift-start').click();
 await expect.poll(()=>samples.filter(s=>s.data.snapshot_kind==='lean-v1').length).toBeGreaterThan(2);
 await page.keyboard.down('KeyD');await page.waitForTimeout(250);await page.keyboard.up('KeyD');await page.keyboard.press('Escape');
 await expect.poll(()=>samples.some(s=>s.data.run?.paused)).toBe(true);
 // Finish body reads before navigation disposes the previous document's responses.
 await page.removeAllListeners('response',{behavior:'wait'});
 const lean=samples.find(s=>s.data.snapshot_kind==='lean-v1'),full=samples.find(s=>s.method==='POST'&&s.data.snapshot_kind==='full'&&s.data.run?.status==='fighting');
 expect(lean.data.run.build).toBeUndefined();expect(lean.data.run.encounter_plan).toBeUndefined();expect(lean.data.run.player).toBeTruthy();expect(full.data.run.build).toBeTruthy();expect(lean.bytes).toBeLessThan(full.bytes);
 require('node:fs').writeFileSync(info.outputPath('ordinary-payload.json'),JSON.stringify({full:full.bytes,lean:lean.bytes}));
 await info.attach('ordinary-payload-sizes',{body:JSON.stringify({full:full.bytes,lean:lean.bytes}),contentType:'application/json'});
 const reloadResponse=page.waitForResponse(r=>new URL(r.url()).pathname==='/api/abyss/rift'&&r.request().method()==='GET');await page.reload();expect((await(await reloadResponse).json()).snapshot_kind).toBe('full');await expect(page.locator('#rift-start')).toBeEnabled();expect(errors).toEqual([]);
});
test('wrong lean baseline stops combat and reload recovers',async({page})=>{
 await page.goto('/abyss/rift?scenario=spawn-hazards');await expect(page.locator('#rift-start')).toBeEnabled();let corrupted=false;
 await page.route('**/api/abyss/rift',async route=>{const response=await route.fetch();const data=await response.json();if(!corrupted&&data.snapshot_kind==='lean-v1'){corrupted=true;data.snapshot_base='0'.repeat(64);}await route.fulfill({response,json:data});});
 await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay-kicker')).toHaveText('CONNECTION PAUSED');expect(corrupted).toBe(true);
 await page.unroute('**/api/abyss/rift');await page.locator('#rift-start').click();await expect(page.locator('#rift-start')).toHaveText('Resume expedition');await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();await page.keyboard.press('Escape');
});
