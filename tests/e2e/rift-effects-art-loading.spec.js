const {test,expect}=require('@playwright/test');

test('idle preview needs no effects atlas',async({page})=>{
 const paths=[];page.on('request',r=>{if(r.url().includes('/rift_effects.png'))paths.push(r.url());});
 await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();
 expect(paths).toEqual([]);
});

for(const saved of [false,true])test((saved?'saved effects':'new encounter')+' waits for effects decoding before play',async({page})=>{
 await page.addInitScript(()=>{
  const decode=HTMLImageElement.prototype.decode;window.effectsDecodeCount=0;
  const gate=new Promise(resolve=>window.releaseEffectsDecode=resolve);
  HTMLImageElement.prototype.decode=function(){if(!this.src.includes('/rift_effects.png'))return decode.call(this);window.effectsDecodeCount++;return gate.then(()=>decode.call(this));};
 });
 await page.goto('/abyss/rift'+(saved?'?scenario=checkpoint':''),{waitUntil:'domcontentloaded'});
 if(!saved){await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('#rift-start').click();}
 await expect.poll(()=>page.evaluate(()=>window.effectsDecodeCount)).toBe(1);
 await expect(page.locator('#rift-overlay')).toBeVisible();await expect(page.locator('#rift-start')).toBeDisabled();
 await page.evaluate(()=>window.releaseEffectsDecode());
 if(saved)await expect(page.locator('#rift-start')).toBeEnabled();else await expect(page.locator('#rift-overlay')).toBeHidden();
 expect(await page.evaluate(()=>window.effectsDecodeCount)).toBe(1);
});

test('invalid effects dimensions reject and concurrent retry requests share a fresh URL',async({page})=>{
 await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();
 let fail=true;const urls=[];
 await page.route('**/static/rift_effects.png*',route=>{urls.push(route.request().url());return fail?route.fulfill({contentType:'image/svg+xml',body:'<svg xmlns="http://www.w3.org/2000/svg" width="1" height="1"/>'}):route.continue();});
 const error=await page.evaluate(async()=>{try{await RiftRenderer.prepareRun({level:{id:1,region:0}});return '';}catch(e){return e.message;}});expect(error).toContain('effects artwork');
 fail=false;await page.evaluate(()=>Promise.all([RiftRenderer.prepareRun({level:{id:1,region:0}}),RiftRenderer.prepareRun({level:{id:1,region:0}})]));
 expect(urls).toHaveLength(2);expect(urls[1]).toMatch(/[?&]retry=1$/);
});

for(const cancel of [false,true])test('reference animation waits for artwork'+(cancel?' and ignores completion after closing':''),async({page})=>{
 await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();
 let release;const gate=new Promise(resolve=>release=resolve);let requests=0;
 await page.route('**/static/rift_effects.png*',async route=>{requests++;await gate;await route.continue();});
 await page.locator('#rift-loadout-preview').click();const entry=page.locator('#rift-glossary-entries article').first();
 await entry.locator('.rift-skill-animation').click();await expect.poll(()=>requests).toBe(1);
 await expect(entry.locator('canvas')).toBeHidden();await expect(entry.locator('.rift-skill-animation-status')).toContainText('Loading');
 if(cancel)await page.locator('#rift-skill-glossary > summary').click();
 release();await page.evaluate(()=>RiftRenderer.prepareEffects());
 if(cancel){await page.locator('#rift-loadout-preview').click();await expect(entry.locator('canvas')).toBeHidden();}
 else{await expect(entry.locator('canvas')).toBeVisible();await expect(entry.locator('.rift-skill-animation-status')).toContainText('Animation complete');}
});

test('reference artwork failure offers replay and recovers without an expedition',async({page})=>{
 await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();
 let fail=true;const urls=[];await page.route('**/static/rift_effects.png*',route=>{urls.push(route.request().url());return fail?route.abort():route.continue();});
 await page.locator('#rift-loadout-preview').click();const entry=page.locator('#rift-glossary-entries article').first(),button=entry.locator('.rift-skill-animation');
 await button.click();await expect(entry.locator('.rift-skill-animation-status')).toContainText('try again');await expect(entry.locator('canvas')).toBeHidden();await expect(button).toBeEnabled();
 fail=false;await button.click();await expect(entry.locator('.rift-skill-animation-status')).toContainText('Animation complete');expect(urls).toHaveLength(2);expect(urls[1]).toMatch(/[?&]retry=1$/);
 expect((await(await page.request.get('/api/abyss/rift')).json()).run).toBeFalsy();
});
