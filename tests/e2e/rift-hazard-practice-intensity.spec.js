const {test,expect}=require('@playwright/test');
test('hazard intensity applies on start and reset, persists, and isolates campaign',async({page})=>{
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();const campaign=(await(await page.request.get('/api/abyss/rift')).json()).run;
 await page.goto('/abyss/rift?practice=hazard');await expect(page.locator('#rift-start')).toBeEnabled();const select=page.locator('#rift-hazard-intensity');await expect(select).toBeVisible();await select.selectOption('gentle');await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();
 const saved=async()=>(await(await page.request.get('/api/abyss/rift?practice=hazard')).json()).run;
 await expect.poll(async()=>(await saved())?.practice?.hazard_intensity).toBe('gentle');expect((await saved()).practice.arena.hazards[0].period).toBe(5);
 await page.keyboard.press('Escape');await expect(page.locator('#rift-paused-badge')).toHaveText('Paused');
 await page.reload();await expect(page.locator('#rift-start')).toBeEnabled();await expect(select).toHaveValue('gentle');
 for(const [intensity,period] of [['intense',2.8],['standard',3.5]]){await select.selectOption(intensity);await page.locator('#rift-practice-reset').click();await expect.poll(async()=>(await saved()).practice.hazard_intensity).toBe(intensity);expect((await saved()).practice.arena.hazards[0].period).toBe(period);}
 expect((await saved()).gold).toBe(0);expect((await saved()).drops).toEqual([]);expect((await(await page.request.get('/api/abyss/rift')).json()).run).toEqual(campaign);
 await page.setViewportSize({width:390,height:844});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);await page.locator('#rift-hazard-practice-options').screenshot({path:'test-results/hazard-practice-intensity.png'});
});
