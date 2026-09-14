const {test,expect}=require('@playwright/test');

test('confirmed mission history survives pauses and reloads without extra attempts',async({page})=>{
  await page.goto('/abyss/rift?mission=42');await expect(page.locator('#rift-start')).toBeEnabled();
  await page.locator('#rift-start').click();await page.keyboard.press('Escape');
  const read=async()=>(await(await page.request.get('/api/abyss/rift')).json()).run;
  const first=await read();expect(first.mission_history['42']).toMatchObject({attempts:1,completions:0,last_outcome:'active'});
  expect(first.mission_history['42'].last_started_ms).toBeGreaterThan(0);
  await page.reload();await expect(page.locator('#rift-start')).toHaveText('Resume expedition');
  await page.locator('#rift-start').click();await page.keyboard.press('Escape');
  const resumed=await read();expect(resumed.mission_history).toEqual(first.mission_history);
});

test('checkpoint exit records an unfinished attempt and a new expedition retains it',async({page})=>{
  await page.goto('/abyss/rift?scenario=checkpoint');await page.locator('#rift-auto').uncheck();
  await page.locator('#rift-start').click();await page.locator('#rift-exit').click();
  await expect(page.locator('#rift-banked')).toHaveText('30 gold · 1 item');
  const read=async()=>(await(await page.request.get('/api/abyss/rift')).json()).run;
  await expect.poll(async()=>(await read()).mission_history['1'].last_outcome).toBe('exited');
  const exited=await read();expect(exited.mission_history['1']).toMatchObject({attempts:1,completions:0});
  expect(exited.mission_history['1'].best_seconds).toBeUndefined();
  await page.goto('/abyss/rift?mission=2');await expect(page.locator('#rift-start')).toBeEnabled();
  await page.locator('#rift-start').click();await page.keyboard.press('Escape');
  const next=await read();expect(next.mission_history['1']).toEqual(exited.mission_history['1']);
  expect(next.mission_history['2']).toMatchObject({attempts:1,last_outcome:'active'});
});
