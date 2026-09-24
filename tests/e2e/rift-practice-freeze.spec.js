const {test,expect}=require('@playwright/test');
test('practice freeze persists, keeps the player mobile and separates assisted records',async({page})=>{
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();const campaign=(await(await page.request.get('/api/abyss/rift')).json()).run;
 await page.goto('/abyss/rift?practice=boss');await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('#rift-start').click();const freeze=page.locator('#rift-practice-freeze'),read=async()=>(await(await page.request.get('/api/abyss/rift?practice=boss')).json()).run;
 await freeze.click();await expect(freeze).toHaveAttribute('aria-pressed','true');let saved=await read();expect(saved.paused).toBe(true);expect(saved.practice.freeze_used).toBe(true);const enemy={x:saved.enemies[0].x,y:saved.enemies[0].y};
 await page.locator('#rift-start').click();await page.keyboard.down('KeyD');await expect.poll(async()=>(await read()).player.x).toBeGreaterThan(saved.player.x+20);await page.keyboard.up('KeyD');await page.keyboard.press('Escape');saved=await read();expect({x:saved.enemies[0].x,y:saved.enemies[0].y}).toEqual(enemy);
 await page.reload();await expect(freeze).toHaveAttribute('aria-pressed','true');await freeze.click();await expect(freeze).toHaveAttribute('aria-pressed','false');saved=await read();expect(saved.practice.freeze_used).toBe(true);
 await page.evaluate(run=>{run.status='complete';run.practice.completed=true;run.stats.seconds=10;RiftRecords.update(run);},saved);await expect(page.locator('#rift-practice-best')).toContainText('Best (freeze assisted):');
 await page.evaluate(run=>{run.status='fighting';run.practice.completed=false;run.practice.freeze_used=false;RiftRecords.update(run);},saved);await expect(page.locator('#rift-practice-best')).toContainText('No completed');
 await page.locator('#rift-practice-reset').click();await expect.poll(async()=>(await read()).practice.freeze_used||false).toBe(false);expect((await read()).practice.freeze_movement||false).toBe(false);
 await page.setViewportSize({width:390,height:1600});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);expect((await(await page.request.get('/api/abyss/rift')).json()).run).toEqual(campaign);
});
