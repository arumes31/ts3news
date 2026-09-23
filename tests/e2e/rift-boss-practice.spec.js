const {test,expect}=require('@playwright/test');
test('select and restart a boss phase without changing campaign',async({page})=>{
 const errors=[];page.on('pageerror',error=>errors.push(error.message));
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
 const campaign=(await(await page.request.get('/api/abyss/rift')).json()).run;
 await page.goto('/abyss/rift?practice=boss');await expect(page.locator('#rift-start')).toBeEnabled();
 const boss=page.locator('#rift-practice-boss'),phase=page.locator('#rift-practice-phase');
 const names=await boss.locator('option').evaluateAll(options=>options.map(o=>o.value));expect(names.length).toBeGreaterThan(1);
 await boss.selectOption(names[1]);await phase.selectOption('3');await page.locator('#rift-start').click();
 const saved=async()=>(await(await page.request.get('/api/abyss/rift?practice=boss')).json()).run;
 await expect.poll(async()=>(await saved())?.practice?.boss_start?.phase).toBe(3);
 expect((await saved()).enemies[0].name).toBe(names[1]);await expect(page.locator('#rift-skills')).toBeVisible();
 await page.keyboard.press('Escape');await expect(page.locator('#rift-paused-badge')).toHaveText('Paused');
 await page.reload();await expect(page.locator('#rift-start')).toHaveText('Resume drill');await expect(boss).toHaveValue(names[1]);await expect(phase).toHaveValue('3');
 await boss.selectOption(names[0]);await phase.selectOption('2');await page.locator('#rift-practice-reset').click();
 await expect.poll(async()=>(await saved()).practice.boss_start.phase).toBe(2);
 const reset=await saved();expect(reset.enemies[0].name).toBe(names[0]);expect(reset.enemies[0].hp).toBe(reset.enemies[0].max_hp*.5);expect(reset.drops).toEqual([]);expect(reset.gold).toBe(0);
 await page.setViewportSize({width:390,height:844});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
 expect((await(await page.request.get('/api/abyss/rift')).json()).run).toEqual(campaign);expect(errors).toEqual([]);
});

test('reset waits for an in-flight boss input request',async({page})=>{
 await page.goto('/abyss/rift?practice=boss');await expect(page.locator('#rift-start')).toBeEnabled();
 let release,observed;const pending=new Promise(resolve=>observed=resolve);const held=new Promise(resolve=>release=resolve);let steps=0;
 await page.route('**/api/abyss/rift?practice=boss',async route=>{
  if(route.request().method()==='POST'&&route.request().postDataJSON().kind==='step'&&++steps===2){observed();await held;}
  await route.continue();
 });
 await page.locator('#rift-start').click();await pending;
 await page.locator('#rift-practice-phase').selectOption('3');
 await page.locator('#rift-practice-reset').click();
 release();
 const saved=async()=>(await(await page.request.get('/api/abyss/rift?practice=boss')).json()).run;
 await expect.poll(async()=>(await saved()).practice.boss_start.phase).toBe(3);
 await expect(page.locator('#rift-start')).toBeVisible();
 await expect(page.locator('#rift-start')).not.toHaveAttribute('data-recover','true');
 await expect(page.locator('#rift-status')).not.toContainText('aborted');
 await page.screenshot({path:'test-results/boss-practice-controls.png',fullPage:true});
});
