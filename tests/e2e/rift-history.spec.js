const {test,expect}=require('@playwright/test');

test('campaign history distinguishes outcomes and sorts confirmed records with stable ties',async({page})=>{
  await page.goto('/abyss/rift?scenario=history');await expect(page.locator('#rift-start')).toBeEnabled();
  await expect(page.locator('#rift-history-1')).toHaveText('3 recorded attempts · Finished · Best 80.0s');
  await expect(page.locator('#rift-history-2')).toHaveText('2 recorded attempts · Defeated');
  await expect(page.locator('#rift-history-3')).toHaveText('1 recorded attempt · Left early');
  await expect(page.locator('#rift-history-6')).toHaveText('Completed before attempt tracking');
  await expect(page.locator('#rift-history-7')).toBeHidden();
  const order=()=>page.locator('#rift-levels [data-level]:visible').evaluateAll(cards=>cards.map(card=>Number(card.dataset.level)));
  await page.locator('#rift-mission-sort').selectOption('best');
  expect((await order()).slice(0,5)).toEqual([4,5,1,2,3]);
  await page.locator('#rift-mission-sort').selectOption('recent');
  expect((await order()).slice(0,5)).toEqual([2,3,4,5,1]);
  await page.reload();await expect(page.locator('#rift-mission-sort')).toHaveValue('recent');
  expect((await order()).slice(0,5)).toEqual([2,3,4,5,1]);
  await page.locator('#rift-completion').selectOption('complete');
  await page.locator('#rift-mission-sort').selectOption('best');expect(await order()).toEqual([4,5,1,6]);
  await page.locator('[data-level="4"]').focus();await page.keyboard.press('ArrowRight');await expect(page.locator('[data-level="5"]')).toBeFocused();
  await page.keyboard.press('Enter');await expect(page.locator('#rift-start')).toHaveText('Enter mission 5');
  await page.locator('#rift-last-attempt').click();await expect(page.locator('#rift-start')).toHaveText('Enter mission 2');
  await expect(page.locator('#rift-completion')).toHaveValue('complete');await expect(page.locator('#rift-selected-mission')).toContainText('Selected mission: 2');
  await page.setViewportSize({width:390,height:844});await page.locator('#rift-compact').check();expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
  await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();await page.keyboard.press('Escape');await page.locator('#rift-campaign > summary').click();
  await expect(page.locator('#rift-last-attempt')).toBeDisabled();
  await page.locator('#rift-clear-filters').click();await expect(page.locator('#rift-history-2')).toContainText('3 recorded attempts · In progress');
});

test('confirmed mission history survives pauses and reloads without extra attempts',async({page})=>{
  await page.goto('/abyss/rift?mission=42');await expect(page.locator('#rift-start')).toBeEnabled();
  await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();await page.keyboard.press('Escape');
  const read=async()=>(await(await page.request.get('/api/abyss/rift')).json()).run;
  const first=await read();expect(first.mission_history['42']).toMatchObject({attempts:1,completions:0,last_outcome:'active'});
  expect(first.mission_history['42'].last_started_ms).toBeGreaterThan(0);
  await page.reload();await expect(page.locator('#rift-start')).toHaveText('Resume expedition');
  await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();await page.keyboard.press('Escape');
  const resumed=await read();expect(resumed.mission_history).toEqual(first.mission_history);
});

test('checkpoint exit records an unfinished attempt and a new expedition retains it',async({page})=>{
  await page.goto('/abyss/rift?scenario=checkpoint');await page.locator('#rift-auto').uncheck();
  await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();await page.locator('#rift-exit').click();
  await expect(page.locator('#rift-banked')).toHaveText('30 gold · 1 item');
  const read=async()=>(await(await page.request.get('/api/abyss/rift')).json()).run;
  await expect.poll(async()=>(await read()).mission_history['1'].last_outcome).toBe('exited');
  const exited=await read();expect(exited.mission_history['1']).toMatchObject({attempts:1,completions:0});
  expect(exited.mission_history['1'].best_seconds).toBeUndefined();
  await page.goto('/abyss/rift?mission=2');await expect(page.locator('#rift-start')).toBeEnabled();
  await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();await page.keyboard.press('Escape');
  const next=await read();expect(next.mission_history['1']).toEqual(exited.mission_history['1']);
  expect(next.mission_history['2']).toMatchObject({attempts:1,last_outcome:'active'});
});
