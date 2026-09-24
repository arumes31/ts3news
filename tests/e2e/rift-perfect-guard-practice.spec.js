const {test,expect}=require('@playwright/test');
test('perfect-guard drill rejects held blocks and resets without campaign rewards',async({page})=>{
 await page.goto('/abyss/rift?scenario=checkpoint');const campaign=(await(await page.request.get('/api/abyss/rift')).json()).run;
 await page.locator('#rift-practice-links a[href$="practice=perfect_guard"]').click();await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();
 await expect(page.locator('#rift-practice-instructions')).toContainText('0.22 seconds');const read=async()=>(await(await page.request.get('/api/abyss/rift?practice=perfect_guard')).json()).run;
 await page.keyboard.down('KeyL');await expect.poll(async()=>(await read()).stats.guards,{timeout:15000}).toBeGreaterThanOrEqual(3);await page.keyboard.up('KeyL');await page.keyboard.press('Escape');
 const held=await read();expect(held.practice.completed).toBe(false);expect(held.practice.perfect_guards||0).toBeLessThan(3);expect(held.stats.perfect_guards).toBe(0);await expect(page.locator('#rift-practice-progress')).toContainText('/3 perfect guards');
 await page.locator('#rift-practice-reset').click();await expect.poll(async()=>(await read()).stats.guards).toBe(0);expect((await read()).practice.perfect_guards||0).toBe(0);
 await page.setViewportSize({width:390,height:1600});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);expect((await(await page.request.get('/api/abyss/rift')).json()).run).toEqual(campaign);
});
