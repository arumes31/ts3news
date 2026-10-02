const {test,expect}=require('@playwright/test');

for(const committed of [false,true])test('reload recovers '+(committed?'a committed pause with a lost reply':'an uncommitted pause')+' without replaying mutations',async({page})=>{
  await page.goto('/abyss/rift');
  await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();
  let release,held=false,posts=0;
  const gate=new Promise(resolve=>release=resolve);
  await page.route('**/api/abyss/rift',async route=>{
    if(route.request().method()!=='POST')return route.continue();
    posts++;
    if(route.request().postDataJSON().kind!=='pause')return route.continue();
    if(committed)await route.fetch();
    held=true;await gate;
    await route.abort('failed').catch(()=>{});
  });
  await page.keyboard.press('Escape');await expect.poll(()=>held).toBe(true);
  const read=async()=>(await(await page.request.get('/api/abyss/rift')).json()).run;
  const saved=await read();expect(!!saved.paused).toBe(committed);
  const sent=posts;
  await page.reload();release();
  await expect(page.locator('#rift-start')).toHaveText('Resume expedition');
  if(committed)await expect(page.locator('#rift-overlay-copy')).toContainText('last confirmed moment');
  else await expect(page.locator('#rift-overlay-copy')).toContainText('pause was not confirmed');
  await page.waitForTimeout(500);
  const recovered=await read();
  expect(posts).toBe(sent);
  expect(recovered.revision).toBe(saved.revision);
  expect(recovered.stats.seconds).toBe(saved.stats.seconds);
  expect(recovered.player.hp).toBe(saved.player.hp);
  expect(recovered.banked_gold).toBe(saved.banked_gold);
  expect(recovered.banked_items).toEqual(saved.banked_items);
  expect(recovered.pause_started_ms).toBe(saved.pause_started_ms);
  await page.unroute('**/api/abyss/rift');
  await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();
  await expect.poll(async()=>(await read()).revision).toBeGreaterThan(saved.revision);
});

test('pause transport allows best-effort delivery during dismissal',async({page})=>{
  await page.addInitScript(()=>{
    const original=window.fetch;window.pauseTransport=[];
    window.fetch=function(url,options){
      if(options?.method==='POST'&&JSON.parse(options.body).kind==='pause')window.pauseTransport.push(options.keepalive===true);
      return original.call(this,url,options);
    };
  });
  await page.goto('/abyss/rift');await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();
  await page.evaluate(()=>window.dispatchEvent(new PageTransitionEvent('pagehide')));
  await expect.poll(()=>page.evaluate(()=>window.pauseTransport)).toEqual([true]);
  await expect.poll(async()=>(await(await page.request.get('/api/abyss/rift')).json()).run.paused).toBe(true);
});

test('history restoration waits for the pause queued behind an input save',async({page})=>{
  await page.goto('/abyss/rift');await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();
  let releaseStep,releasePause,stepHeld=false,pauseHeld=false,reads=0;
  const stepGate=new Promise(resolve=>releaseStep=resolve),pauseGate=new Promise(resolve=>releasePause=resolve);
  await page.route('**/api/abyss/rift',async route=>{
    if(route.request().method()==='GET'){reads++;return route.continue();}
    const kind=route.request().postDataJSON().kind;
    if(kind==='step'&&!stepHeld){stepHeld=true;await stepGate;}
    if(kind==='pause'){pauseHeld=true;await pauseGate;}
    await route.continue();
  });
  await expect.poll(()=>stepHeld).toBe(true);
  await page.evaluate(()=>{
    window.dispatchEvent(new PageTransitionEvent('pagehide',{persisted:true}));
    window.dispatchEvent(new PageTransitionEvent('pageshow',{persisted:true}));
  });
  releaseStep();await expect.poll(()=>pauseHeld).toBe(true);
  expect(reads).toBe(0);
  releasePause();
  await expect(page.locator('#rift-overlay-title')).toHaveText('Your expedition awaits.');
  expect(reads).toBe(1);
  expect((await(await page.request.get('/api/abyss/rift')).json()).run.paused).toBe(true);
});
