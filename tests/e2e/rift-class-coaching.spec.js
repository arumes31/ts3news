const {test,expect}=require('@playwright/test');

test('finisher statistics display and class coaching stays dismissed after reload',async({page})=>{
  await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
  const display=async empty=>page.evaluate(async empty=>{const run=(await(await fetch('/api/abyss/rift')).json()).run;run.stats.empty_finishers=empty;run.stats.charged_finishers=4;run.stats.charges_spent=10;window.RiftHUD.update(run,false,true);},empty);
  await display(2);await expect(page.locator('#rift-class-coaching')).toBeHidden();
  await display(3);await expect(page.locator('#rift-class-coaching')).toBeVisible();
  await expect(page.locator('#rift-class-coaching')).toContainText('Iron Guard (Q) before Resolute Bash (E)');
  await page.locator('.rift-run-statistics > summary').click();
  const value=label=>page.locator('#rift-statistics dt').filter({hasText:label}).locator('xpath=following-sibling::dd[1]');
  await expect(value('Charged finishers')).toHaveText('4');await expect(value('Finishers without charges')).toHaveText('3');await expect(value('Charges spent')).toHaveText('10');
  await page.setViewportSize({width:390,height:844});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
  await page.getByRole('button',{name:'Dismiss class coaching'}).click();await expect(page.locator('#rift-class-coaching')).toBeHidden();
  await page.reload();await expect(page.locator('#rift-start')).toBeEnabled();await display(5);await expect(page.locator('#rift-class-coaching')).toBeHidden();
});

test('real empty and charged finishers update persisted sequence counters',async({page})=>{
  await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();
  const read=async()=>(await(await page.request.get('/api/abyss/rift')).json()).run;
  await page.keyboard.press('e');await expect.poll(async()=>(await read()).stats.empty_finishers).toBe(1);
  await page.waitForTimeout(400);await page.keyboard.press('q');await expect.poll(async()=>(await read()).resource).toBe(1);
  await expect(page.locator('[data-ability-role="finisher"]')).toBeEnabled({timeout:15000});await page.keyboard.press('e');
  await expect.poll(async()=>(await read()).stats.charged_finishers).toBe(1);expect((await read()).stats.charges_spent).toBe(1);
  const spent=(await read()).last_charge_spend;expect(spent.charges).toBe(1);expect(spent.skill_name).toBe('Resolute Bash');await expect(page.locator('#rift-last-charge-spend')).toHaveText('Last charge spend: Resolute Bash consumed 1 charge.');
  await page.keyboard.press('Escape');await page.reload();await expect(page.locator('#rift-start')).toBeEnabled();await expect(page.locator('#rift-last-charge-spend')).toHaveText('Last charge spend: Resolute Bash consumed 1 charge.');const restored=await read();expect(restored.last_charge_spend).toEqual(spent);expect(restored.stats.empty_finishers).toBe(1);expect(restored.stats.charged_finishers).toBe(1);
});

test('charge spend protocol supports legacy snapshots and rejects malformed spends',async({page})=>{
 await page.goto('/abyss/rift?scenario=spawn-hazards');await expect(page.locator('#rift-start')).toBeEnabled();const data=await(await page.request.get('/api/abyss/rift')).json();
 expect(await page.evaluate(data=>[undefined,{skill_id:'a',skill_name:'A',charges:3},{skill_id:'a',skill_name:'A',charges:0},{skill_id:'a',skill_name:'A',charges:4},{skill_id:'a',skill_name:'A',charges:1.5},{skill_id:'a',skill_name:7,charges:1},null].map(spend=>{const value=structuredClone(data);value.run.last_charge_spend=spend;try{RiftProtocol.validate(value,'GET');return true;}catch(_){return false;}}),data)).toEqual([true,true,false,false,false,false,false]);
 await expect(page.locator('#rift-last-charge-spend')).toBeHidden();
});
