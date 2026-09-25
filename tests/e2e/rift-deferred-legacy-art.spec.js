const {test,expect}=require('@playwright/test');
test('campaign startup does not fetch unused legacy boss art',async({page})=>{
 const requests=[];page.on('request',r=>{if(r.url().includes('rift_boss_area.png'))requests.push(r.url());});
 await page.goto('/abyss/rift?scenario=visual');await expect(page.locator('#rift-start')).toBeEnabled();
 expect(requests).toEqual([]);
});
test('saved legacy boss scene waits for its art and reuses the decoded image',async({page})=>{
 let release;const held=new Promise(resolve=>release=resolve);let requested=0;
 await page.route('**/static/rift_boss_area.png*',async route=>{requested++;await held;await route.continue();});
 await page.route('**/api/abyss/rift',async route=>{const response=await route.fetch();const data=await response.json();delete data.run.level;data.run.room=2;await route.fulfill({response,json:data});});
 await page.goto('/abyss/rift?scenario=visual',{waitUntil:'domcontentloaded'});
 await expect.poll(()=>requested).toBe(1);await expect(page.locator('#rift-start')).toBeDisabled();
 release();await expect(page.locator('#rift-start')).toBeEnabled();
 await page.evaluate(async()=>{await RiftRenderer.prepareRun({room:2});await RiftRenderer.prepareRun({room:2});});
 expect(requested).toBe(1);
});

test('failed deferred artwork can retry without a cached rejection',async({page})=>{
 await page.goto('/abyss/rift?scenario=visual');await expect(page.locator('#rift-start')).toBeEnabled();
 let attempts=0;
 await page.route('**/static/rift_boss_area.png*',async route=>{attempts++;if(attempts===1)await route.abort();else await route.continue();});
 const failure=await page.evaluate(()=>RiftRenderer.prepareRun({room:2}).then(()=>'',error=>error.message));
 expect(failure).toContain('Could not load saved boss scene artwork');
 await page.evaluate(()=>Promise.all([RiftRenderer.prepareRun({room:2}),RiftRenderer.prepareRun({room:2})]));
 expect(attempts).toBe(2);
});
