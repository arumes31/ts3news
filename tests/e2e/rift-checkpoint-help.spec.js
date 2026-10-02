const {test,expect}=require('@playwright/test');

test('checkpoint help follows seamless setting and remembers dismissals separately',async({page})=>{
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();await expect(page.locator('#rift-checkpoint-guide')).toBeVisible();await expect(page.locator('#rift-checkpoint-guide-copy')).toContainText('Seamless tiers');
 await page.locator('#rift-auto').uncheck();await expect(page.locator('#rift-checkpoint-guide-copy')).toContainText('Manual checkpoints');await expect(page.locator('#rift-checkpoint-guide-copy')).toContainText('Bank & leave');await page.locator('#rift-checkpoint-guide-dismiss').click();await expect(page.locator('#rift-checkpoint-guide')).toBeHidden();
 await page.reload();await expect(page.locator('#rift-start')).toBeEnabled();await expect(page.locator('#rift-checkpoint-guide')).toBeHidden();await page.locator('#rift-auto').check();await expect(page.locator('#rift-checkpoint-guide-copy')).toContainText('Seamless tiers');await page.locator('#rift-checkpoint-guide-dismiss').click();await page.reload();await expect(page.locator('#rift-start')).toBeEnabled();await expect(page.locator('#rift-checkpoint-guide')).toBeHidden();
});

test('checkpoint explanation remains readable after seamless advancement',async({page})=>{
 await page.goto('/abyss/rift?scenario=checkpoint');await page.locator('#rift-start').click();await page.locator('#rift-next').click();await expect.poll(async()=>(await(await page.request.get('/api/abyss/rift')).json()).run.room).toBe(1);await page.keyboard.press('Escape');await expect(page.locator('#rift-paused-badge')).toHaveText('Paused');await expect(page.locator('#rift-checkpoint-guide')).toBeVisible();await expect(page.locator('#rift-checkpoint-guide-copy')).toContainText('confirmed banking');
 await page.setViewportSize({width:390,height:844});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
});
