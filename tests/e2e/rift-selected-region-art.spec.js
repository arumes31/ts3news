const {test,expect}=require('@playwright/test');
test('initial preview downloads one region panel without the full sheet',async({page})=>{
 const images=[];page.on('request',r=>{if(/rift_region(?:s|_\d)\.png/.test(r.url()))images.push(new URL(r.url()).pathname);});
 await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();
 expect(images).toEqual(['/static/rift_region_0.png']);
});

test('latest mission selection waits for its own region and ignores late preview loads',async({page})=>{
 await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();
 await page.addStyleTag({content:'.rift-level-art{display:none!important}'});
 let release;const gate=new Promise(resolve=>release=resolve);let requests=0;
 await page.route('**/static/rift_region_1.png*',async route=>{requests++;await gate;await route.continue();});
 await page.locator('[data-level="11"]').click();await expect.poll(()=>requests).toBeGreaterThan(0);await expect(page.locator('#rift-start')).toBeDisabled();
 await page.locator('[data-level="21"]').click();await expect(page.locator('#rift-start')).toHaveText('Enter mission 21');await expect(page.locator('#rift-start')).toBeEnabled();
 release();await page.evaluate(()=>RiftRenderer.prepareRegion(1));
 await expect(page.locator('#rift-start')).toHaveText('Enter mission 21');
 await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();await page.locator('#rift-pause').click();
 const run=(await(await page.request.get('/api/abyss/rift')).json()).run;expect(run.level.id).toBe(21);
});

test('failed mission artwork can retry without changing the selected mission',async({page})=>{
 await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();
 await page.addStyleTag({content:'.rift-level-art{display:none!important}'});
 let fail=true;await page.route('**/static/rift_region_4.png*',route=>fail?route.abort():route.continue());
 await page.locator('[data-level="41"]').click();await expect(page.locator('#rift-start')).toHaveText('Retry region artwork');
 fail=false;await page.locator('#rift-start').click();await expect(page.locator('#rift-start')).toHaveText('Enter mission 41');await expect(page.locator('#rift-start')).toBeEnabled();
});

test('saved region gates readiness and concurrent preparation shares its request',async({page})=>{
 let release;const gate=new Promise(resolve=>release=resolve);let requests=0;
 await page.route('**/api/abyss/rift',async route=>{if(route.request().method()!=='GET')return route.continue();const response=await route.fetch(),data=await response.json();data.run.level.region=8;data.run.level.id=81;await route.fulfill({response,json:data});});
 await page.route('**/static/rift_region_8.png*',async route=>{requests++;await gate;await route.continue();});
 await page.goto('/abyss/rift?scenario=visual',{waitUntil:'domcontentloaded'});await expect.poll(()=>requests).toBe(1);await expect(page.locator('#rift-start')).toBeDisabled();
 release();await expect(page.locator('#rift-start')).toBeEnabled();await page.evaluate(()=>Promise.all([RiftRenderer.prepareRegion(8),RiftRenderer.prepareRegion(8)]));expect(requests).toBe(1);
});

test('region boundary prefetch is reused and final mission does not request an eleventh region',async({page})=>{
 await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();
 const paths=[];page.on('request',r=>{if(/rift_region_\d+\.png/.test(r.url()))paths.push(new URL(r.url()).pathname);});
 await page.evaluate(async()=>{await RiftRenderer.prepareRun({status:'fighting',level:{id:10,region:0}});await RiftRenderer.prepareRegion(1);await RiftRenderer.prepareRun({status:'fighting',level:{id:11,region:1}});await RiftRenderer.prepareRun({status:'fighting',level:{id:100,region:9}});});
 expect(paths).toEqual(['/static/rift_region_1.png','/static/rift_region_9.png']);
});

test('saved final checkpoint prepares the next region before continuation',async({page})=>{
 await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();
 let release;const gate=new Promise(resolve=>release=resolve);let requested=0;
 await page.route('**/static/rift_region_1.png*',async route=>{requested++;await gate;await route.continue();});
 await page.evaluate(()=>{window.regionGateDone=false;window.regionGatePromise=RiftRenderer.prepareRun({status:'cleared',room:2,level:{id:10,region:0}}).then(()=>{window.regionGateDone=true;});});
 await expect.poll(()=>requested).toBe(1);expect(await page.evaluate(()=>window.regionGateDone)).toBe(false);
 release();await page.evaluate(()=>window.regionGatePromise);expect(await page.evaluate(()=>window.regionGateDone)).toBe(true);
});

test('mission thumbnails load their region panel when scrolled into view',async({page},info)=>{
 await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();
 const card=page.locator('[data-level="91"]');await card.scrollIntoViewIfNeeded();
 await expect(card.locator('.rift-level-art')).toHaveCSS('background-image',/rift_region_9\.png/);
 await expect(card.locator('.rift-level-art')).toHaveCSS('background-size','cover');
 await page.locator('#rift-campaign').screenshot({path:info.outputPath('region-campaign.png')});
});
