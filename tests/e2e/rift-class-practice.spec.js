const {test,expect}=require('@playwright/test');
for(const subclass of ['vanguard','berserker','marksman','beastmaster','elementalist','chronomancer','oracle','geomancer','bloodblade','voidwalker','runesmith','alchemist'])test('class tutorial uses equipped '+subclass+' sequence without campaign rewards',async({page})=>{
 await page.goto('/abyss/rift?scenario=checkpoint&subclass='+subclass);await expect(page.locator('#rift-start')).toBeEnabled();const campaign=(await(await page.request.get('/api/abyss/rift')).json()).run;
 await page.goto('/abyss/rift?practice=class');await expect(page.locator('#rift-start')).toBeEnabled();await expect(page.locator('#rift-boss-practice-options')).toBeHidden();const {build}=await(await page.request.get('/api/abyss/rift?practice=class')).json();
 for(const skill of build.signatures)await expect(page.locator('#rift-practice-instructions')).toContainText(skill.name);
 await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();await expect(page.locator('#rift-signatures')).toBeVisible();await expect(page.locator('#rift-skills')).toBeVisible();
 const read=async()=>(await(await page.request.get('/api/abyss/rift?practice=class')).json()).run;
 expect((await read()).build.signatures).toEqual(build.signatures);await page.keyboard.press('q');await expect.poll(async()=>(await read()).resource).toBe(1);await expect.poll(async()=>(await read()).player.cooldown).toBe(0);await page.keyboard.press('e');await expect.poll(async()=>(await read()).practice.completed).toBe(true);
 const done=await read();expect(done.practice.class_hits).toBeGreaterThan(0);expect(done.gold).toBe(0);expect(done.drops).toEqual([]);expect((await(await page.request.get('/api/abyss/rift')).json()).run).toEqual(campaign);
 if(subclass==='vanguard'){await page.reload();await expect(page.locator('#rift-practice-progress')).toHaveText('Drill complete');await page.locator('#rift-practice-reset').click();await expect.poll(async()=>(await read()).practice.completed).toBe(false);expect((await read()).build.signatures).toEqual(build.signatures);}
});

test('class practice fits mobile and explains missing signatures before start',async({page})=>{
 await page.goto('/abyss/rift?practice=class');await expect(page.locator('#rift-start')).toBeEnabled();await page.setViewportSize({width:390,height:1200});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);await page.locator('#rift-practice-guide').scrollIntoViewIfNeeded();await page.locator('#rift-practice-guide').screenshot({path:'test-results/class-practice-mobile.png'});
 await page.route('**/api/abyss/rift?practice=class',async route=>{const response=await route.fetch(),data=await response.json();data.build.signatures=[];data.run=null;await route.fulfill({response,json:data});});await page.reload();await expect(page.locator('#rift-start')).toHaveText('Class abilities required');await expect(page.locator('#rift-start')).toBeDisabled();await expect(page.locator('#rift-practice-instructions')).toContainText('Unlock and equip');
});
