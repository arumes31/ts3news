const {test,expect}=require('@playwright/test');
test('active floor hazard fails only the optional goal and persists on reload',async({page})=>{
 await page.goto('/abyss/rift?scenario=hazard-objective');await expect(page.locator('#rift-start')).toBeEnabled();
 const goal=page.locator('#rift-objectives-list [data-objective-id="hazard_avoidance"]');await expect(goal).toContainText('Safe footing');await expect(goal).toContainText('In progress');
 await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();
 const saved=async()=>(await(await page.request.get('/api/abyss/rift')).json()).run;
 await expect.poll(async()=>(await saved()).objectives.entries.find(e=>e.id==='hazard_avoidance').status).toBe('failed');
 await page.keyboard.press('Escape');await expect(page.locator('#rift-paused-badge')).toHaveText('Paused');await expect(goal).toContainText('Triggered an active floor hazard.');expect((await saved()).status).toBe('fighting');
 await page.evaluate(()=>history.replaceState(null,'','/abyss/rift'));await page.reload();await expect(page.locator('#rift-start')).toBeEnabled();await expect(goal).toContainText('Failed');
});
