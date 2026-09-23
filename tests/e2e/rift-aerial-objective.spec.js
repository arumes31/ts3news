const {test,expect}=require('@playwright/test');
test('jump and attack confirm an aerial finish and preserve it on reload',async({page})=>{
 await page.goto('/abyss/rift?scenario=treasure-objective');await expect(page.locator('#rift-start')).toBeEnabled();
 const goal=page.locator('#rift-objectives-list [data-objective-id="aerial_finish"]');await expect(goal).toContainText('0 / 1 aerial finishes');
 await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();
 await page.keyboard.down('Space');await page.keyboard.down('KeyJ');
 const saved=async()=>(await(await page.request.get('/api/abyss/rift')).json()).run;
 try {await expect.poll(async()=>(await saved()).objectives.entries.find(e=>e.id==='aerial_finish').current).toBe(1);}finally{await page.keyboard.up('KeyJ');await page.keyboard.up('Space');}
 await page.keyboard.press('Escape');await expect(page.locator('#rift-paused-badge')).toHaveText('Paused');await expect(goal).toContainText('1 / 1 aerial finishes');await expect(goal).toContainText('In progress');
 await page.evaluate(()=>history.replaceState(null,'','/abyss/rift'));await page.reload();await expect(page.locator('#rift-start')).toBeEnabled();await expect(goal).toContainText('1 / 1 aerial finishes');
});
