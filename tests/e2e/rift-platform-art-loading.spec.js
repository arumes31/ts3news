const {test,expect}=require('@playwright/test');
test('first mission does not download the unused stone platform surface',async({page})=>{
 const requests=[];page.on('request',r=>{if(r.url().includes('rift_platform_surface.png'))requests.push(r.url());});
 await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();
 await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();await page.locator('#rift-pause').click();
 expect(requests).toEqual([]);
});
test('future rooms share one platform decode and wait for its completion',async({page})=>{
 await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();
 let release,requests=0;const gate=new Promise(resolve=>release=resolve);
 await page.route('**/static/rift_platform_surface.png*',async route=>{requests++;await gate;await route.continue();});
 await page.evaluate(()=>{window.platformDone=false;const run={level:{id:1,region:0,rooms:[{}, {platforms:[{}]},{}]}};window.platformPreparation=Promise.all([RiftRenderer.prepareRun(run),RiftRenderer.prepareRun(run)]).then(()=>window.platformDone=true);});
 try{await expect.poll(()=>requests).toBe(1);expect(await page.evaluate(()=>platformDone)).toBe(false);}finally{release();}
 await page.evaluate(()=>platformPreparation);expect(requests).toBe(1);
});
test('platform preview failures can retry and preserve mission selection',async({page})=>{
 await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();
 let fail=true;await page.route('**/static/rift_platform_surface.png*',route=>fail?route.abort():route.continue());
 await page.locator('[data-level="2"]').click();await expect(page.locator('#rift-start')).toHaveText('Retry region artwork');
 fail=false;await page.locator('#rift-start').click();await expect(page.locator('#rift-start')).toHaveText('Enter mission 2');await expect(page.locator('#rift-start')).toBeEnabled();
});
