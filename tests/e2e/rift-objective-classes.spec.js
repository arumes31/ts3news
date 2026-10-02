const {test,expect}=require('@playwright/test');
for(const subclass of ['vanguard','berserker','marksman','beastmaster','elementalist','chronomancer','oracle','geomancer','bloodblade','voidwalker','runesmith','alchemist']){
 test(subclass+' confirms its canonical builder/finisher objective',async({page})=>{
  await page.goto('/abyss/rift?subclass='+subclass);await expect(page.locator('#rift-start')).toBeEnabled();
  const goal=page.locator('#rift-objectives-list [data-objective-id="finisher"]');await expect(goal).toContainText('Class finisher');
  await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();
  const saved=async()=>(await(await page.request.get('/api/abyss/rift')).json()).run;
  await page.keyboard.press('KeyQ');await expect.poll(async()=>(await saved()).resource).toBeGreaterThan(0);await expect.poll(async()=>(await saved()).player.cooldown).toBe(0);
  await page.keyboard.press('KeyE');await expect.poll(async()=>(await saved()).objectives.entries.find(e=>e.id==='finisher').current).toBe(1);
  await page.keyboard.press('Escape');await expect(page.locator('#rift-paused-badge')).toHaveText('Paused');await expect(goal).toContainText('1 / 1 charged finishers');expect((await saved()).build.class).toBe(subclass);
 });
}
