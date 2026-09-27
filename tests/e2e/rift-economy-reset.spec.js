const {test,expect}=require('@playwright/test');

test('an economy reset interrupts a live fight and recovers without old rewards',async({page})=>{
  let posts=0;
  await page.route('**/api/abyss/rift',route=>{if(route.request().method()==='POST')posts++;return route.continue();});
  await page.goto('/abyss/rift?scenario=checkpoint');
  await page.locator('#rift-auto').uncheck();await page.locator('#rift-start').click();await page.locator('#rift-next').click();
  const read=async()=>(await(await page.request.get('/api/abyss/rift')).json()).run;
  await expect.poll(async()=>(await read()).room).toBe(1);
  const old=await read();expect(old.status).toBe('fighting');expect(old.banked_gold).toBeGreaterThan(0);expect(old.banked_items.length).toBeGreaterThan(0);
  expect((await page.request.post('/api/e2e/rift-economy-reset')).ok()).toBe(true);
  await expect(page.locator('#rift-overlay-kicker')).toHaveText('ECONOMY RESET');
  await expect(page.locator('#rift-overlay-copy')).toContainText('Old unbanked loot cannot carry over');
  const stopped=posts;await page.waitForTimeout(300);expect(posts).toBe(stopped);
  await page.locator('#rift-start').click();
  await expect(page.locator('#rift-overlay-kicker')).toHaveText('EXPEDITION EXPIRED');
  expect(posts).toBe(stopped);
  const expired=await read();expect(expired.id).toBe(old.id);expect(expired.status).toBe('expired');
  expect(expired.gold).toBe(0);expect(expired.drops).toEqual([]);expect(expired.banked_gold).toBe(0);expect(expired.banked_items).toEqual([]);
  expect(expired.past_expeditions.gold).toBe((old.past_expeditions?.gold||0)+old.banked_gold);
  expect(expired.mission_history).toEqual(old.mission_history);
  await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();await page.keyboard.press('Escape');
  await expect.poll(async()=>(await read()).paused).toBe(true);
  const fresh=await read();expect(fresh.id).not.toBe(old.id);expect(fresh.epoch).not.toBe(expired.epoch);
  expect(fresh.gold).toBe(0);expect(fresh.banked_gold).toBe(0);expect(fresh.banked_items).toEqual([]);
  expect(fresh.past_expeditions.gold).toBe(expired.past_expeditions.gold);
  expect(fresh.past_expeditions.gear).toBe(expired.past_expeditions.gear);
});
