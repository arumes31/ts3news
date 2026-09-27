const {test,expect}=require('@playwright/test');

for(const mode of ['saved','failed','already-pausing'])test('leaving Brawl: '+mode,async({page})=>{
  const fail=mode==='failed';
  await page.goto('/abyss/rift');
  await page.locator('#rift-start').click();
  let release;
  const held=new Promise(resolve=>release=resolve);
  let pauses=0;
  await page.route('**/api/abyss/rift',async route=>{
    if(route.request().method()!=='POST'||route.request().postDataJSON().kind!=='pause')return route.continue();
    pauses++;await held;
    if(fail)return route.abort('failed');
    await route.continue();
  });
  await page.route('**/inventory',route=>route.fulfill({contentType:'text/html',body:'<h1>Inventory</h1>'}));
  await page.evaluate(()=>{const a=document.createElement('a');a.id='leave-test';a.href='/inventory';a.textContent='Inventory';document.body.prepend(a);});
  if(mode==='already-pausing'){await page.keyboard.press('Escape');await expect.poll(()=>pauses).toBe(1);}
  await page.locator('#leave-test').click();
  await expect.poll(()=>pauses).toBe(1);
  expect(new URL(page.url()).pathname).toBe('/abyss/rift');
  await page.locator('#leave-test').click();
  release();
  if(fail){
    await expect(page.locator('#rift-start')).toHaveAttribute('data-recover','true');
    expect(new URL(page.url()).pathname).toBe('/abyss/rift');
    const run=(await(await page.request.get('/api/abyss/rift')).json()).run;
    expect(run.paused).toBeFalsy();
  }else{
    await expect(page).toHaveURL(/\/inventory$/);
    const run=(await(await page.request.get('/api/abyss/rift')).json()).run;
    expect(run.paused).toBe(true);expect(run.pause_started_ms).toBeGreaterThan(0);
  }
  expect(pauses).toBe(1);
});

test('leaving during start waits for start and then saves a pause',async({page})=>{
  await page.goto('/abyss/rift');
  let release,starts=0,pauses=0;
  const held=new Promise(resolve=>release=resolve);
  await page.route('**/api/abyss/rift',async route=>{
    if(route.request().method()==='POST'){
      const kind=route.request().postDataJSON().kind;
      if(kind==='start'){starts++;await held;}
      if(kind==='pause')pauses++;
    }
    await route.continue();
  });
  await page.route('**/inventory',route=>route.fulfill({contentType:'text/html',body:'<h1>Inventory</h1>'}));
  await page.evaluate(()=>{const a=document.createElement('a');a.id='leave-test';a.href='/inventory';a.textContent='Inventory';document.body.prepend(a);});
  await page.locator('#rift-start').click();
  await expect.poll(()=>starts).toBe(1);
  await page.locator('#leave-test').click();
  expect(new URL(page.url()).pathname).toBe('/abyss/rift');expect(pauses).toBe(0);
  release();
  await expect(page).toHaveURL(/\/inventory$/);
  const run=(await(await page.request.get('/api/abyss/rift')).json()).run;
  expect(run.paused).toBe(true);expect(pauses).toBe(1);
});
