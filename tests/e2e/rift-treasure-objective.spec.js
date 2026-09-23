const {test,expect}=require('@playwright/test');
test('treasure capture follows a real attack and persists across reload',async({page})=>{
 await page.goto('/abyss/rift?scenario=treasure-objective');await expect(page.locator('#rift-start')).toBeEnabled();
 const goal=page.locator('#rift-objectives-list [data-objective-id="treasure_capture"]');await expect(goal).toContainText('Treasure hunter');await expect(goal).toContainText('0 / 1 goblins captured');
 await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();await page.keyboard.press('KeyJ');
 const saved=async()=>(await(await page.request.get('/api/abyss/rift')).json()).run;
 await expect.poll(async()=>(await saved()).objectives.entries.find(e=>e.id==='treasure_capture').current).toBe(1);
 await page.keyboard.press('Escape');await expect(page.locator('#rift-paused-badge')).toHaveText('Paused');await expect(goal).toContainText('1 / 1 goblins captured');await expect(goal).toContainText('In progress');
 await page.evaluate(()=>history.replaceState(null,'','/abyss/rift'));await page.reload();await expect(page.locator('#rift-start')).toBeEnabled();await expect(goal).toContainText('1 / 1 goblins captured');
});
