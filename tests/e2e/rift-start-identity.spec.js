const {test,expect}=require('@playwright/test');

for(const reload of [false,true])for(const committed of [false,true])test('uncertain start '+(committed?'committed':'uncommitted')+' recovers through '+(reload?'reload':'read')+' with one identity',async({page})=>{
  const starts=[];let interrupted=false;
  await page.route('**/api/abyss/rift',async route=>{
    if(route.request().method()!=='POST'||route.request().postDataJSON().kind!=='start')return route.continue();
    starts.push(route.request().postDataJSON());
    if(!interrupted){interrupted=true;if(committed)await route.fetch();return route.abort('failed');}
    return route.continue();
  });
  await page.goto('/abyss/rift');await page.locator('#rift-start').click();
  await expect(page.locator('#rift-start')).toHaveAttribute('data-recover','true');
  expect(starts).toHaveLength(1);
  if(reload)await page.reload();else await page.locator('#rift-start').click();
  await expect(page.locator('#rift-start')).toBeEnabled();
  if(committed)await expect(page.locator('#rift-start')).toHaveText('Resume expedition');
  await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();
  await page.keyboard.press('Escape');
  const read=async()=>(await(await page.request.get('/api/abyss/rift')).json()).run;
  await expect.poll(async()=>(await read()).paused).toBe(true);
  const run=await read();expect(run.start_key).toBe(starts[0].request_id);expect(run.mission_history['1'].attempts).toBe(1);
  if(committed)expect(starts).toHaveLength(1);else{expect(starts).toHaveLength(2);expect(starts[1]).toEqual(starts[0]);}
  expect(run.banked_gold).toBe(0);expect(run.banked_items).toEqual([]);
  expect(await page.evaluate(()=>sessionStorage.getItem('riftPendingStart:campaign'))).toBeNull();
});

for(const changed of [false,true])test(changed?'changing mission after failed start creates a new intent':'same-page start retry retains identity when session storage is blocked',async({page})=>{
  if(!changed)await page.addInitScript(()=>{
    const get=Storage.prototype.getItem,set=Storage.prototype.setItem,remove=Storage.prototype.removeItem;
    Storage.prototype.getItem=function(key){if(this===sessionStorage)throw new Error('blocked');return get.call(this,key);};
    Storage.prototype.setItem=function(key,value){if(this===sessionStorage)throw new Error('blocked');return set.call(this,key,value);};
    Storage.prototype.removeItem=function(key){if(this===sessionStorage)throw new Error('blocked');return remove.call(this,key);};
  });
  const starts=[];
  await page.route('**/api/abyss/rift',route=>{
    if(route.request().method()!=='POST'||route.request().postDataJSON().kind!=='start')return route.continue();
    starts.push(route.request().postDataJSON());if(starts.length===1)return route.abort('failed');return route.continue();
  });
  await page.goto('/abyss/rift');await page.locator('#rift-start').click();await expect(page.locator('#rift-start')).toHaveAttribute('data-recover','true');
  await page.locator('#rift-start').click();await expect(page.locator('#rift-start')).toBeEnabled();
  if(changed){await page.locator('#rift-campaign').evaluate(node=>node.open=true);await page.locator('#rift-levels [data-level="2"]').click();}
  await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();
  expect(starts).toHaveLength(2);
  if(changed){expect(starts[1].request_id).not.toBe(starts[0].request_id);expect(starts[1].level_id).toBe(2);}
  else expect(starts[1]).toEqual(starts[0]);
});
