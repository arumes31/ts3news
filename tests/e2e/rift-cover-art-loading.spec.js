const {test,expect}=require('@playwright/test');
test('mission one does not request unused terrain cover artwork',async({page})=>{
 const paths=[];page.on('request',r=>{if(r.url().includes('rift_terrain_cover.png'))paths.push(r.url());});
 await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();
 await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();await page.locator('#rift-pause').click();expect(paths).toEqual([]);
});
test('saved future cover and practice arenas wait for one shared decode',async({page})=>{
 await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();
 let release,requests=0;const gate=new Promise(resolve=>release=resolve);
 await page.route('**/static/rift_terrain_cover.png*',async route=>{requests++;await gate;await route.continue();});
 await page.evaluate(()=>{window.coverDone=false;window.coverPreparation=Promise.all([
  RiftRenderer.prepareRun({level:{id:1,region:0,rooms:[{}, {}, {cover:[{}]}]}}),
  RiftRenderer.prepareRun({level:{id:1,region:0},practice:{arena:{cover:[{}]}}})
 ]).then(()=>window.coverDone=true);});
 try{await expect.poll(()=>requests).toBe(1);expect(await page.evaluate(()=>coverDone)).toBe(false);}finally{release();}
 await page.evaluate(()=>coverPreparation);expect(requests).toBe(1);
});
test('failed cover preview retries without switching the selected mission',async({page})=>{
 await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();
 let fail=true;await page.route('**/static/rift_terrain_cover.png*',route=>fail?route.abort():route.continue());
 await page.locator('[data-level="4"]').click();await expect(page.locator('#rift-start')).toHaveText('Retry region artwork');
 fail=false;await page.locator('#rift-start').click();await expect(page.locator('#rift-start')).toHaveText('Enter mission 4');await expect(page.locator('#rift-start')).toBeEnabled();
});
