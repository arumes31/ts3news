const {test,expect}=require('@playwright/test');
test('idle combo expires and the next real attack starts at one',async({page})=>{
 await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('#rift-start').click();
 const current=async()=>(await(await page.request.get('/api/abyss/rift')).json()).run;
 await page.keyboard.down('j');
 await expect.poll(async()=>(await current()).combo).toBeGreaterThanOrEqual(2);
 await page.keyboard.up('j');
 const peak=(await current()).stats.highest_combo;
 await expect.poll(async()=>(await current()).combo,{timeout:5000}).toBe(0);
 expect((await current()).stats.highest_combo).toBe(peak);
 await page.keyboard.down('j');
 await expect.poll(async()=>(await current()).combo,{intervals:[30,50,50],timeout:2000}).toBe(1);
 await page.keyboard.up('j');
 await page.keyboard.press('Escape');await expect(page.locator('#rift-paused-badge')).toHaveText('Paused');
 const paused=await current();await page.waitForTimeout(1400);
 expect((await current()).combo_time).toBe(paused.combo_time);
});
