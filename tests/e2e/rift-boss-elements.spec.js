const {test,expect}=require('@playwright/test');

test('boss practice shows the saved current ward across phase reset and recovery',async({page},info)=>{
 const errors=[];page.on('pageerror',error=>errors.push(error.message));
 await page.goto('/abyss/rift?practice=boss');await expect(page.locator('#rift-start')).toBeEnabled();
 const data=await(await page.request.get('/api/abyss/rift?practice=boss')).json();
 const boss=data.bestiary.find(enemy=>enemy.kind==='boss'&&enemy.elemental_phases?.length===3);expect(boss).toBeTruthy();
 await page.locator('#rift-practice-boss').selectOption(boss.name);await page.locator('#rift-practice-phase').selectOption('2');
 await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();await page.keyboard.press('Escape');await expect(page.locator('#rift-paused-badge')).toHaveText('Paused');
 const read=async()=>(await(await page.request.get('/api/abyss/rift?practice=boss')).json()).run;
 const hint=page.locator('#rift-boss-ward');
 const expected=phase=>{const ward=boss.elemental_phases[phase-1];return ward.element+' ward · '+ward.weakness+' direct hits ×2 before defenses';};
 await expect(hint).toHaveText(expected(2));
 await page.locator('#rift-practice-phase').selectOption('3');await page.locator('#rift-practice-reset').click();
 await expect.poll(async()=>(await read()).enemies[0].phase).toBe(3);await expect(hint).toHaveText(expected(3));
 await page.reload();await expect(page.locator('#rift-start')).toHaveText('Resume drill');await expect(hint).toHaveText(expected(3));
 await page.setViewportSize({width:390,height:844});await page.locator('#rift-start').click();await expect(hint).toBeVisible();await page.locator('#rift-viewport').evaluate(el=>el.scrollIntoView({block:'center'}));await hint.screenshot({path:info.outputPath('ward-mobile.png')});const box=await hint.boundingBox(),viewport=await page.locator('#rift-viewport').boundingBox();expect(box.y).toBeGreaterThanOrEqual(viewport.y);expect(box.y+box.height).toBeLessThanOrEqual(viewport.y+viewport.height);await page.locator('#rift-viewport').screenshot({path:info.outputPath('boss-mobile.png')});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
 await page.keyboard.press('Escape');await expect(page.locator('#rift-paused-badge')).toHaveText('Paused');
 const saved=await read();await page.evaluate(run=>{delete run.enemies[0].elemental_phases;RiftHUD.update(run,false);},saved);await expect(hint).toHaveAttribute('hidden','');
 await page.evaluate(run=>{run.enemies=[];RiftHUD.update(run,false);},saved);await expect(hint).toHaveAttribute('hidden','');expect(errors).toEqual([]);
});
