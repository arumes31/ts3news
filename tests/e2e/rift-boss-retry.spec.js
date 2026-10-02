const {test,expect}=require('@playwright/test');
test('retry restores the defeated boss room and keeps banked rewards',async({page})=>{
 await page.goto('/abyss/rift?scenario=boss-retry');await expect(page.locator('#rift-start')).toBeEnabled();
 const saved=async()=>(await(await page.request.get('/api/abyss/rift')).json()).run;
 const before=await saved();expect(before.status).toBe('defeated');
 await page.locator('#rift-retry-boss').click();await expect(page.locator('#rift-overlay-title')).toHaveText('Boss room ready.');
 const after=await saved();expect(after.id).toBe(before.id);expect(after.room).toBe(2);expect(after.banked_gold).toBe(50);expect(after.gold).toBe(0);expect(after.drops).toEqual([]);expect(after.paused).toBe(true);expect(after.player.hp).toBe(after.player.max_hp);expect(after.encounter_plan).toEqual(before.encounter_plan);
 await page.evaluate(()=>history.replaceState(null,'','/abyss/rift'));await page.reload();await expect(page.locator('#rift-start')).toBeEnabled();
 expect((await saved()).room).toBe(2);await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();
 await expect.poll(async()=>(await saved()).paused).toBe(false);
});
