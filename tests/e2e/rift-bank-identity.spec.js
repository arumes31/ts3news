const {test,expect}=require('@playwright/test');

for(const kind of ['exit','next','advance'])for(const reload of [false,true])test('uncommitted '+kind+' keeps its identity after '+(reload?'reload':'read'),async({page})=>{
  const banks=[];
  await page.route('**/api/abyss/rift',route=>{
    if(route.request().method()!=='POST'||route.request().postDataJSON().kind!==kind)return route.continue();
    banks.push(route.request().postDataJSON());if(banks.length===1)return route.abort('failed');return route.continue();
  });
  await page.goto('/abyss/rift?scenario=checkpoint');await page.evaluate(()=>history.replaceState(null,'','/abyss/rift'));
  await page.locator('#rift-auto').setChecked(kind==='advance');await page.locator('#rift-start').click();
  const button=kind==='exit'?'#rift-exit':'#rift-next';await page.locator(button).click();
  await expect(page.locator('#rift-start')).toHaveText('Recover expedition');expect(banks).toHaveLength(1);
  if(reload)await page.reload();else await page.locator('#rift-start').click();
  await expect(page.locator('#rift-start')).toHaveText('Resume expedition');await page.waitForTimeout(200);expect(banks).toHaveLength(1);
  await page.locator('#rift-start').click();await page.locator(button).click();
  const read=async()=>(await(await page.request.get('/api/abyss/rift')).json()).run;
  await expect.poll(async()=>(await read()).banked_gold).toBe(30);
  expect(banks).toHaveLength(2);expect(banks[1].request_id).toBe(banks[0].request_id);
  expect(banks[1].revision).toBeGreaterThan(banks[0].revision);
  expect({...banks[1],revision:banks[0].revision}).toEqual(banks[0]);
  if(kind!=='exit'){await page.keyboard.press('Escape');await expect.poll(async()=>(await read()).paused).toBe(true);}
  const saved=await read();expect(saved.banked_items).toHaveLength(1);
  const late=await page.request.post('/api/abyss/rift',{data:banks[0]});expect(late.ok()).toBe(true);
  const after=await read();expect(after.revision).toBe(saved.revision);expect(after.banked_gold).toBe(30);expect(after.banked_items).toEqual(saved.banked_items);
  expect(await page.evaluate(()=>sessionStorage.getItem('riftPendingBank:campaign'))).toBeNull();
});

for(const recovery of ['read','reload','changed'])test(recovery==='changed'?'choosing a different checkpoint action creates a new bank intent':'confirmed exit clears its identity after a lost response and '+recovery,async({page})=>{
  const changed=recovery==='changed';
  const banks=[];let failed=false;
  await page.route('**/api/abyss/rift',async route=>{
    const request=route.request();
    if(request.method()!=='POST'||!['exit','next'].includes(request.postDataJSON().kind))return route.continue();
    banks.push(request.postDataJSON());
    if(!failed){failed=true;if(!changed)await route.fetch();return route.abort('failed');}
    return route.continue();
  });
  await page.goto('/abyss/rift?scenario=checkpoint');await page.evaluate(()=>history.replaceState(null,'','/abyss/rift'));
  await page.locator('#rift-auto').uncheck();await page.locator('#rift-start').click();await page.locator('#rift-exit').click();
  await expect(page.locator('#rift-start')).toHaveText('Recover expedition');
  if(recovery==='reload')await page.reload();else await page.locator('#rift-start').click();
  if(changed){
    await expect(page.locator('#rift-start')).toHaveText('Resume expedition');await page.locator('#rift-start').click();await page.locator('#rift-next').click();
    await expect.poll(()=>banks.length).toBe(2);expect(banks[1].kind).toBe('next');expect(banks[1].request_id).not.toBe(banks[0].request_id);
  }else{
    await expect(page.locator('#rift-overlay-title')).toHaveText('Returned from the ruins.');await page.waitForTimeout(250);expect(banks).toHaveLength(1);
    expect(await page.evaluate(()=>sessionStorage.getItem('riftPendingBank:campaign'))).toBeNull();
  }
  const run=(await(await page.request.get('/api/abyss/rift')).json()).run;expect(run.banked_gold).toBe(30);expect(run.banked_items).toHaveLength(1);
});

test('another confirmed bank in the same cleared tier invalidates the old pending intent',async({page})=>{
  const exits=[];
  await page.route('**/api/abyss/rift',route=>{
    if(route.request().method()==='POST'&&route.request().postDataJSON().kind==='exit'){
      exits.push(route.request().postDataJSON());if(exits.length===1)return route.abort('failed');
    }
    return route.continue();
  });
  await page.goto('/abyss/rift?scenario=checkpoint');await page.locator('#rift-auto').uncheck();
  await page.locator('#rift-start').click();await page.locator('#rift-exit').click();await expect(page.locator('#rift-start')).toHaveText('Recover expedition');
  const response=await page.request.post('/api/abyss/rift',{data:{...exits[0],kind:'bank',request_id:'other-confirmed-bank-request'}});expect(response.ok()).toBe(true);
  expect((await response.json()).run.status).toBe('cleared');
  await page.locator('#rift-start').click();await expect(page.locator('#rift-start')).toHaveText('Resume expedition');
  expect(await page.evaluate(()=>sessionStorage.getItem('riftPendingBank:campaign'))).toBeNull();
  await page.locator('#rift-start').click();await page.locator('#rift-exit').click();await expect(page.locator('#rift-overlay-title')).toHaveText('Returned from the ruins.');
  expect(exits).toHaveLength(2);expect(exits[1].request_id).not.toBe(exits[0].request_id);
  const run=(await(await page.request.get('/api/abyss/rift')).json()).run;expect(run.banked_gold).toBe(30);expect(run.banked_items).toHaveLength(1);
});
