const {test,expect}=require('@playwright/test');
test('free practice selects live monsters, saves them, and clears without campaign changes',async({page})=>{
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();const campaign=(await(await page.request.get('/api/abyss/rift')).json()).run;
 await page.goto('/abyss/rift?practice=skills');await expect(page.locator('#rift-start')).toBeEnabled();const data=await(await page.request.get('/api/abyss/rift?practice=skills')).json();
 const select=page.locator('#rift-practice-enemy');await expect(select).toBeVisible();expect(await select.locator('option').evaluateAll(nodes=>nodes.map(n=>n.value))).toEqual(data.bestiary.map(m=>m.name));
 await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();
 const read=async()=>(await(await page.request.get('/api/abyss/rift?practice=skills')).json()).run;
 for(const unit of [data.bestiary.find(m=>m.kind!=='boss'),data.bestiary.find(m=>m.kind==='boss')]){
  await select.selectOption(unit.name);await page.locator('[data-practice-action="practice_spawn"]').click();await expect.poll(async()=>(await read()).enemies.map(e=>e.name)).toEqual([unit.name]);expect((await read()).paused).toBe(true);await expect(page.locator('#rift-practice-tool-status')).toContainText(unit.name);
 }
 const chosen=(await read()).enemies[0];await page.reload();await expect(page.locator('#rift-start')).toBeEnabled();expect((await read()).enemies[0]).toEqual(chosen);await expect(select).toHaveValue(chosen.name);
 await page.locator('[data-practice-action="practice_clear"]').click();await expect.poll(async()=>(await read()).enemies.length).toBe(0);expect((await read()).projectiles).toEqual([]);await page.reload();await expect(page.locator('#rift-start')).toBeEnabled();expect((await read()).enemies).toEqual([]);
 await page.locator('#rift-practice-reset').click();await expect.poll(async()=>(await read()).enemies[0]?.id).toBe('practice-target');expect((await(await page.request.get('/api/abyss/rift')).json()).run).toEqual(campaign);
 await page.setViewportSize({width:390,height:1200});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
 await page.goto('/abyss/rift?practice=combo');await expect(page.locator('#rift-start')).toBeEnabled();await expect(page.locator('#rift-practice-enemy-controls')).toBeHidden();
});
