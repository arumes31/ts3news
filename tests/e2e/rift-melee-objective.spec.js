const {test,expect}=require('@playwright/test');
test('ranged cast fails melee-only objective without ending combat and persists',async({page})=>{
 await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();
 const goal=page.locator('#rift-objectives-list [data-objective-id="melee_only"]');await expect(goal).toContainText('Close combat');await expect(goal).toContainText('heals and shields');
 await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();await page.keyboard.press('Digit1');
 const saved=async()=>(await(await page.request.get('/api/abyss/rift')).json()).run;
 await expect.poll(async()=>(await saved()).stats.skill_uses?.guard||0).toBe(1);await expect(goal).toContainText('In progress');await expect.poll(async()=>(await saved()).player.cooldown).toBe(0);await page.keyboard.press('Digit3');
 await expect.poll(async()=>(await saved()).objectives.entries.find(e=>e.id==='melee_only').status).toBe('failed');
 await page.keyboard.press('Escape');await expect(page.locator('#rift-paused-badge')).toHaveText('Paused');await expect(goal).toContainText('1 non-melee casts');expect((await saved()).status).toBe('fighting');
 await page.reload();await expect(page.locator('#rift-start')).toBeEnabled();await expect(goal).toContainText('Failed');await expect(goal).toContainText('Cast an ability outside the melee/support allowance.');
});
