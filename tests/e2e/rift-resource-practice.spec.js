const {test,expect}=require('@playwright/test');
for(const subclass of ['vanguard','berserker','marksman','beastmaster','elementalist','chronomancer','oracle','geomancer','bloodblade','voidwalker','runesmith','alchemist'])test('resource drill completes with '+subclass+' and preserves campaign',async({page})=>{
 test.setTimeout(90000);
 await page.goto('/abyss/rift?scenario=checkpoint&subclass='+subclass);await expect(page.locator('#rift-start')).toBeEnabled();const campaign=(await(await page.request.get('/api/abyss/rift')).json()).run;
 await page.locator('#rift-practice-links a[href$="practice=resource"]').click();await expect(page.locator('#rift-start')).toBeEnabled();const {build}=await(await page.request.get('/api/abyss/rift?practice=resource')).json();expect(build.class).toBe(subclass);for(const skill of build.signatures)await expect(page.locator('#rift-practice-instructions')).toContainText(skill.name);
 await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();const read=async()=>(await(await page.request.get('/api/abyss/rift?practice=resource')).json()).run;
 for(let cycle=1;cycle<=2;cycle++){
  for(let charge=1;charge<=3;charge++){await expect(page.locator('[data-ability-role="builder"]')).toBeEnabled({timeout:15000});await page.keyboard.press('q');await expect.poll(async()=>(await read()).resource).toBe(charge);}
  await expect(page.locator('[data-ability-role="finisher"]')).toBeEnabled({timeout:15000});await page.keyboard.press('e');await expect.poll(async()=>(await read()).practice.resource_cycles).toBe(cycle);
 }
 await expect(page.locator('#rift-practice-progress')).toHaveText('Drill complete');const done=await read();expect(done.build.signatures).toEqual(build.signatures);expect(done.stats.charges_spent).toBe(6);expect(done.stats.empty_finishers).toBe(0);expect(done.gold).toBe(0);expect(done.drops).toEqual([]);
 await page.reload();await expect(page.locator('#rift-practice-progress')).toHaveText('Drill complete');await page.locator('#rift-practice-reset').click();await expect.poll(async()=>(await read()).practice.resource_cycles||0).toBe(0);expect((await(await page.request.get('/api/abyss/rift')).json()).run).toEqual(campaign);
 await page.setViewportSize({width:390,height:1200});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
});

test('resource drill validates saved progress and explains missing abilities',async({page})=>{
 await page.goto('/abyss/rift?practice=resource');await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();await page.keyboard.press('Escape');const data=await(await page.request.get('/api/abyss/rift?practice=resource')).json();
 expect(await page.evaluate(data=>[undefined,0,1,2,-1,3,1.5,'1',null].map(value=>{const copy=structuredClone(data);copy.run.practice.resource_cycles=value;try{RiftProtocol.validate(copy,'GET');return true;}catch(_){return false;}}),data)).toEqual([true,true,true,true,false,false,false,false,false]);
 await page.locator('#rift-controls-open').click();await page.locator('[data-remap="signature0"]').click();await page.keyboard.press('h');await page.locator('#rift-controls-close').click();await expect(page.locator('#rift-practice-instructions')).toContainText('Iron Guard (H)');
 await page.route('**/api/abyss/rift?practice=resource',async route=>{const response=await route.fetch(),data=await response.json();data.build.signatures=[];data.run=null;await route.fulfill({response,json:data});});await page.reload();await expect(page.locator('#rift-start')).toHaveText('Class abilities required');await expect(page.locator('#rift-start')).toBeDisabled();await expect(page.locator('#rift-practice-instructions')).toContainText('Unlock and equip');
});
