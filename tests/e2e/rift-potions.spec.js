const {test,expect}=require('@playwright/test');

test('owned potion heals once and preserves inventory, cooldown and objective on recovery',async({page},info)=>{
 const errors=[];page.on('pageerror',error=>errors.push(error.message));
 await page.goto('/abyss/rift?scenario=potions');
 await expect(page.locator('#rift-start')).toHaveText('Resume expedition');
 await page.locator('#rift-auto').uncheck();
 const saved=async()=>(await(await page.request.get('/api/abyss/rift')).json()).run;
 const inventory=async()=>(await(await page.request.get('/api/abyss/rift?inventory=potions')).json()).potions;
 await page.locator('#rift-potions summary').click();
 await expect(page.locator('#rift-potion-select')).toContainText('Small Health Potion');
 const use=page.locator('#rift-potion-use');await expect(use).toBeDisabled();
 expect((await inventory())[0].count).toBe(2);
 await page.locator('#rift-start').click();await expect(use).toBeEnabled();await use.click();
 await expect.poll(async()=>(await saved()).stats.potions_used).toBe(1);
 await page.keyboard.press('Escape');await expect(page.locator('#rift-paused-badge')).toHaveText('Paused');
 const before=await saved();expect(before.player.hp).toBe(150);expect(before.skill_timers.healing_potion).toBeGreaterThan(0);
 expect(before.objectives.entries.find(goal=>goal.id==='no_potions').status).toBe('failed');
 await expect(page.locator('#rift-objectives-list [data-objective-id="no_potions"]')).toHaveAttribute('data-state','failed');
 await expect(page.locator('#rift-objectives-list [data-objective-id="no_potions"]')).toContainText('1 potion used');
 expect((await inventory())[0].count).toBe(1);await expect(use).toBeDisabled();
 await page.evaluate(()=>history.replaceState(null,'','/abyss/rift'));await page.reload();
 await expect(page.locator('#rift-start')).toHaveText('Resume expedition');
 const recovered=await saved();expect(recovered.player.hp).toBe(150);expect(recovered.stats.potions_used).toBe(1);
 expect(recovered.skill_timers.healing_potion).toBe(before.skill_timers.healing_potion);
 expect(recovered.objectives.entries.find(goal=>goal.id==='no_potions').status).toBe('failed');
 await page.locator('#rift-potions summary').click();await expect(page.locator('#rift-potion-select')).toContainText('Small Health Potion');
 expect((await inventory())[0].count).toBe(1);
 await page.setViewportSize({width:390,height:844});await page.locator('#rift-potions').screenshot({path:info.outputPath('potions-mobile.png')});
 expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);expect(errors).toEqual([]);
});

test('practice never exposes shared potion inventory',async({page})=>{
 await page.goto('/abyss/rift?practice=boss');await expect(page.locator('#rift-start')).toBeEnabled();
 await expect(page.locator('#rift-potions')).toBeHidden();
 const result=await(await page.request.get('/api/abyss/rift?practice=boss&inventory=potions')).json();expect(result.potions).toEqual([]);
});
