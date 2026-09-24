const {test,expect}=require('@playwright/test');
test('skill animation previews replay, stop, respect reduced motion and preserve expedition state',async({page})=>{
 await page.goto('/abyss/rift?scenario=spawn-hazards&subclass=elementalist');await expect(page.locator('#rift-start')).toBeEnabled();
 const before=(await(await page.request.get('/api/abyss/rift')).json()).run;await page.locator('#rift-loadout-preview').click();
 const entries=page.locator('#rift-glossary-entries article'),first=entries.first(),canvas=first.locator('canvas');
 await expect(canvas).toBeHidden();const audioBefore=await page.evaluate(()=>RiftAudio.played);
 await first.locator('.rift-skill-animation').click();await expect(canvas).toBeVisible();await expect(first.locator('.rift-skill-animation-status')).toContainText('Animation complete');await expect(canvas).toHaveAttribute('data-frame','5');
 expect(await page.evaluate(()=>RiftAudio.played)).toBe(audioBefore);
 const last=entries.last();await last.locator('.rift-skill-animation').click();await expect(canvas).toBeHidden();await expect(last.locator('canvas')).toBeVisible();
 await page.locator('#rift-skill-glossary > summary').click();const stopped=await last.locator('canvas').getAttribute('data-frame');await page.waitForTimeout(850);expect(await last.locator('canvas').getAttribute('data-frame')).toBe(stopped);
 await page.locator('#rift-loadout-preview').click();await page.evaluate(()=>RiftRenderer.reduced=true);await first.locator('.rift-skill-animation').click();await expect(canvas).toHaveAttribute('data-frame','2');await expect(first.locator('.rift-skill-animation-status')).toContainText('Static preview');
 await page.setViewportSize({width:390,height:1600});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);await first.scrollIntoViewIfNeeded();await page.evaluate(()=>window.scrollBy(0,-240));await first.screenshot({path:'test-results/skill-animation-mobile.png'});
 expect((await(await page.request.get('/api/abyss/rift')).json()).run).toEqual(before);
});
