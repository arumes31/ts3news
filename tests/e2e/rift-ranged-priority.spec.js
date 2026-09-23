const {test,expect}=require('@playwright/test');
test('defeating melee enemy before a live archer fails the priority goal',async({page})=>{
 await page.goto('/abyss/rift?scenario=ranged-priority');await expect(page.locator('#rift-start')).toBeEnabled();
 const goal=page.locator('#rift-objectives-list [data-objective-id="ranged_priority"]');await expect(goal).toContainText('Ranged enemies first');await expect(goal).toContainText('In progress');
 await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();await page.keyboard.press('KeyJ');
 const saved=async()=>(await(await page.request.get('/api/abyss/rift')).json()).run;
 await expect.poll(async()=>(await saved()).objectives.entries.find(e=>e.id==='ranged_priority').status).toBe('failed');
 await page.keyboard.press('Escape');await expect(page.locator('#rift-paused-badge')).toHaveText('Paused');await expect(goal).toContainText('1 priority violations');expect((await saved()).status).toBe('fighting');
 await page.evaluate(()=>history.replaceState(null,'','/abyss/rift'));await page.reload();await expect(page.locator('#rift-start')).toBeEnabled();await expect(goal).toContainText('while a ranged enemy was still alive');
});
