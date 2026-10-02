const {test,expect}=require('@playwright/test');
test('ultimate timing completes in an opening and preserves campaign',async({page})=>{
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();const campaign=(await(await page.request.get('/api/abyss/rift')).json()).run;
 await page.locator('#rift-practice-links a[href$="practice=ultimate"]').click();await expect(page.locator('#rift-start')).toBeEnabled();
 await expect(page.locator('#rift-practice-instructions')).toContainText('Rift Nova (R)');
 await page.locator('#rift-controls-open').click();await page.locator('[data-remap="ultimate"]').click();await page.keyboard.press('h');await page.locator('#rift-controls-close').click();await expect(page.locator('#rift-practice-instructions')).toContainText('Rift Nova (H)');
 await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();
 await expect(page.locator('#rift-practice-progress')).toContainText('Prepare');
 await expect(page.locator('#rift-objective')).toContainText('Opening active',{timeout:10000});await page.keyboard.press('KeyH');
 const read=async()=>(await(await page.request.get('/api/abyss/rift?practice=ultimate')).json()).run;
 await expect.poll(async()=>(await read()).practice.completed).toBe(true);
 await expect(page.locator('#rift-practice-progress')).toHaveText('Drill complete');
 const completed=await read();expect(completed.practice.ultimate_timed).toBe(true);expect(completed.stats.ultimate_casts).toBe(1);expect(completed.gold).toBe(0);expect(completed.drops).toHaveLength(0);
 await page.reload();await expect(page.locator('#rift-practice-progress')).toHaveText('Drill complete');await page.locator('#rift-practice-reset').click();
 await expect.poll(async()=>(await read()).practice.ultimate_timed||false).toBe(false);
 await page.setViewportSize({width:390,height:1600});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
 expect((await(await page.request.get('/api/abyss/rift')).json()).run).toEqual(campaign);
});
test('ultimate drill explains missing equipment',async({page})=>{
 await page.route('**/api/abyss/rift?practice=ultimate',async route=>{const response=await route.fetch(),data=await response.json();delete data.build.ultimate;data.run=null;await route.fulfill({response,json:data});});
 await page.goto('/abyss/rift?practice=ultimate');await expect(page.locator('#rift-start')).toHaveText('Ultimate required');await expect(page.locator('#rift-start')).toBeDisabled();await expect(page.locator('#rift-practice-instructions')).toContainText('Equip an ultimate in Abyss');
});
