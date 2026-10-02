const {test,expect}=require('@playwright/test');
test('preview and track a charged class finisher through real controls',async({page})=>{
 await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();
 const objective=page.locator('#rift-objectives-list [data-objective-id="finisher"]');await expect(objective).toContainText('Empty finishers do not count');
 await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();
 const saved=async()=>(await(await page.request.get('/api/abyss/rift')).json()).run;
 await page.keyboard.press('KeyQ');await expect.poll(async()=>(await saved()).resource).toBeGreaterThan(0);await expect.poll(async()=>(await saved()).player.cooldown).toBe(0);
 await page.keyboard.press('KeyE');await expect.poll(async()=>(await saved()).objectives.entries.find(e=>e.id==='finisher').current).toBe(1);
 await page.keyboard.press('Escape');await expect(page.locator('#rift-paused-badge')).toHaveText('Paused');await expect(objective).toContainText('1 / 1 charged finishers');await expect(objective).toContainText('In progress');
 await page.reload();await expect(page.locator('#rift-start')).toBeEnabled();await expect(objective).toContainText('1 / 1 charged finishers');
});
