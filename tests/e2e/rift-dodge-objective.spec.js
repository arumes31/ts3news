const {test,expect}=require('@playwright/test');
test('jumping over four real shots exceeds the dodge allowance and persists',async({page})=>{
 await page.goto('/abyss/rift?scenario=dodge-objective');await expect(page.locator('#rift-start')).toBeEnabled();
 const goal=page.locator('#rift-objectives-list [data-objective-id="limited_dodge"]');await expect(goal).toContainText('0 / 3 allowed dodges');
 await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();await page.keyboard.press('Space');
 const saved=async()=>(await(await page.request.get('/api/abyss/rift')).json()).run;
 await expect.poll(async()=>(await saved()).objectives.entries.find(e=>e.id==='limited_dodge').status).toBe('failed');
 await page.keyboard.press('Escape');await expect(page.locator('#rift-paused-badge')).toHaveText('Paused');await expect(goal).toContainText('4 / 3 allowed dodges');expect((await saved()).status).toBe('fighting');expect((await saved()).stats.damage_taken).toBe(0);
 await page.evaluate(()=>history.replaceState(null,'','/abyss/rift'));await page.reload();await expect(page.locator('#rift-start')).toBeEnabled();await expect(goal).toContainText('Exceeded three airborne dodges');
});
