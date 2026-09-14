const {test,expect}=require('@playwright/test');

test('finished mission results open region progress and replay the finished mission once',async({page})=>{
  await page.goto('/abyss/rift?scenario=checkpoint&room=final');await page.locator('#rift-auto').uncheck();
  await page.locator('#rift-start').click();await page.locator('#rift-next').click();
  await expect(page.locator('#rift-replay')).toBeVisible();await expect(page.locator('#rift-replay')).toHaveText('Replay mission 1');
  await page.locator('#rift-result-region').click();
  await expect(page.locator('#rift-campaign')).toHaveAttribute('open','');
  await expect(page.locator('[data-overview-region="0"] details')).toHaveAttribute('open','');
  await expect(page.locator('[data-overview-region="0"] summary')).toBeFocused();
  await expect(page.locator('[data-overview-region="0"] summary')).toContainText('1/10 completed');
  const read=async()=>(await(await page.request.get('/api/abyss/rift')).json()).run;
  const finished=await read();expect(finished.status).toBe('complete');
  const starts=[];page.on('request',request=>{if(request.url().endsWith('/api/abyss/rift')&&request.method()==='POST'&&request.postDataJSON().kind==='start')starts.push(request.postDataJSON());});
  await page.locator('#rift-replay').evaluate(button=>{button.click();button.click();});
  await expect(page.locator('#rift-overlay')).toBeHidden();await page.keyboard.press('Escape');
  const replay=await read();expect(replay.id).not.toBe(finished.id);expect(replay.level.id).toBe(1);expect(replay.room).toBe(0);
  expect(replay.mission_history['1'].attempts).toBe(2);expect(replay.completed_levels).toContain(1);expect(starts).toHaveLength(1);
  await expect(page.locator('#rift-result-actions')).toBeHidden();
});

test('early exits offer region progress without claiming a completed mission replay',async({page})=>{
  await page.goto('/abyss/rift?scenario=checkpoint');await page.locator('#rift-auto').uncheck();
  await page.locator('#rift-start').click();await page.locator('#rift-exit').click();
  await expect(page.locator('#rift-result-region')).toBeVisible();await expect(page.locator('#rift-replay')).toBeHidden();
  await page.setViewportSize({width:390,height:844});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
  await page.locator('#rift-result-region').click();await expect(page.locator('[data-overview-region="0"] summary')).toContainText('0/10 completed');
});
