const { test, expect } = require('@playwright/test');

test('empty regular loadout still supports class combat', async ({ page }) => {
  await page.goto('/abyss/rift?subclass=geomancer');
  for(const select of await page.locator('#rift-loadout select').all())await select.selectOption('');
  await page.locator('#rift-start').click();
  await expect(page.locator('#rift-overlay')).toBeHidden();
  const run=(await(await page.request.get('/api/abyss/rift')).json()).run;
  expect(run.build.skills).toEqual([]);
  await page.keyboard.down('q');
  await expect.poll(async()=>(await(await page.request.get('/api/abyss/rift')).json()).run.barrier).toBeGreaterThan(0);
  await page.keyboard.up('q');
  await page.keyboard.press('Escape');
});

test('all Abyss subclasses build and spend their own resource', async ({ page }) => {
  test.setTimeout(120_000);
  const errors=[]; page.on('pageerror',error=>errors.push(error.message));
  for(const style of ['vanguard','berserker','marksman','beastmaster','elementalist','chronomancer','oracle','geomancer','bloodblade','voidwalker','runesmith','alchemist']){
    await page.goto('/abyss/rift?subclass='+style);
    await page.locator('#rift-start').click();
    await expect(page.locator('#rift-overlay')).toBeHidden();
    await page.keyboard.down('q');
    await expect.poll(async()=>(await(await page.request.get('/api/abyss/rift')).json()).run.resource).toBe(1);
    await page.keyboard.up('q');
    await page.keyboard.down('e');
    await expect.poll(async()=>(await(await page.request.get('/api/abyss/rift')).json()).run.resource).toBe(0);
    await page.keyboard.up('e');
    const run=(await(await page.request.get('/api/abyss/rift')).json()).run;
    expect(run.build.class).toBe(style);
    expect(run.skill_timers[run.build.signatures[1].id]).toBeGreaterThan(0);
    await page.keyboard.press('Escape');
  }
  expect(errors).toEqual([]);
});

test('clear all three rooms, defeat Thornheart and bank the expedition', async ({ page }) => {
  test.setTimeout(180_000);
  const errors=[];page.on('pageerror',error=>errors.push(error.message));
  await page.goto('/abyss/rift?subclass=bloodblade');
  await page.locator('#rift-start').click();
  const held=new Set();
  async function controls(wanted){for(const key of [...held])if(!wanted.has(key)){await page.keyboard.up(key);held.delete(key);}for(const key of wanted)if(!held.has(key)){await page.keyboard.down(key);held.add(key);}}
  const started=Date.now();let complete=false;
  while(Date.now()-started<145_000){
    const run=(await(await page.request.get('/api/abyss/rift')).json()).run;
    expect(run.status,'expedition should remain survivable').not.toBe('defeated');
    if(run.status==='complete'){complete=true;expect(run.banked_gold).toBe(300);expect(run.banked_items.length).toBeGreaterThanOrEqual(4);break;}
    if(run.status==='cleared'){
      await controls(new Set());
      await expect(page.locator('#rift-next')).toBeVisible();
      if(run.room===2)await page.screenshot({path:'test-results/rift-boss-cleared.png'});
      await page.locator('#rift-next').click();continue;
    }
    const p=run.player,target=run.enemies.filter(e=>e.hp>0).sort((a,b)=>Math.abs(a.x-p.x)-Math.abs(b.x-p.x))[0];
    const dx=target.x-p.x,dy=target.y-p.y,wanted=new Set(['k']);
    if(Math.abs(dy)>10)wanted.add(dy>0?'s':'w');
    if(Math.abs(dx)>60 || Math.sign(dx)!==p.facing)wanted.add(dx>0?'d':'a');
    if(Math.abs(dx)<120&&Math.abs(dy)<30){
      const [builder,finisher]=run.build.signatures;
      if(run.resource>0&&!(run.skill_timers[finisher.id]>0)&&p.mana>=finisher.cost)wanted.add('e');
      else if(!(run.skill_timers[builder.id]>0)&&p.mana>=builder.cost)wanted.add('q');
      else wanted.add('j');
    }
    await controls(wanted);
    await page.waitForTimeout(120);
  }
  await controls(new Set());
  expect(complete).toBe(true);expect(errors).toEqual([]);
  await expect(page.locator('#rift-overlay-title')).toHaveText('Returned from the ruins.');
});

test('sprite stage, keyboard combat, audio and pause recovery', async ({ page }) => {
  const errors = [];
  page.on('pageerror', error => errors.push(error.message));
  await page.goto('/abyss/rift');
  await expect(page.locator('#rift-start')).toHaveText('Enter the ruins →');
  await expect(page.locator('#rift-build')).toContainText('Rowan');
  await page.locator('#rift-start').click();
  await expect(page.locator('#rift-overlay')).toBeHidden();
  const initial = (await (await page.request.get('/api/abyss/rift')).json()).run;
  await page.keyboard.down('d');
  await expect.poll(async () => (await (await page.request.get('/api/abyss/rift')).json()).run.player.x).toBeGreaterThan(initial.player.x + 40);
  await page.keyboard.up('d');
  await page.keyboard.down('3');
  await expect.poll(async () => (await (await page.request.get('/api/abyss/rift')).json()).run.skill_timers.spark || 0).toBeGreaterThan(0);
  await page.keyboard.up('3');
  await expect.poll(() => page.evaluate(() => window.RiftAudio.played)).toBeGreaterThan(0);
  await expect.poll(() => page.evaluate(() => window.RiftAudio.context.state)).toBe('running');
  await page.keyboard.press('Escape');
  await expect(page.locator('#rift-overlay-title')).toHaveText('A moment by the lantern.');
  await expect.poll(() => page.evaluate(() => window.RiftAudio.context.state)).toBe('suspended');
  const paused = (await (await page.request.get('/api/abyss/rift')).json()).run;
  await page.reload();
  await expect(page.locator('#rift-start')).toHaveText('Resume expedition');
  const saved = (await (await page.request.get('/api/abyss/rift')).json()).run;
  expect(saved.id).toBe(paused.id);
  expect(saved.player.hp).toBe(paused.player.hp);
  await page.screenshot({ path: 'test-results/rift-paused.png' });
  expect(errors).toEqual([]);
});

test('checkpoint banks actual catalog loot once and survives reload', async ({ page }) => {
  await page.goto('/abyss/rift?scenario=checkpoint');
  await expect(page.locator('#rift-start')).toHaveText('Resume expedition');
  await page.locator('#rift-start').click();
  await expect(page.locator('#rift-loot')).not.toContainText('Your next discovery');
  const before = (await (await page.request.get('/api/abyss/rift')).json()).run;
  expect(before.drops[0].gear.ID).toMatch(/^ABYSS_/);
  await page.locator('#rift-exit').click();
  await expect(page.locator('#rift-banked')).toHaveText('30 gold · 1 items');
  const banked = (await (await page.request.get('/api/abyss/rift')).json()).run;
  await page.request.post('/api/abyss/rift', {data:{kind:'exit',run_id:banked.id,revision:banked.revision,request_id:'retry-bank-request',input:{}}});
  await page.goto('/abyss/rift');
  await expect(page.locator('#rift-banked')).toHaveText('30 gold · 1 items');
});

test('mobile touch controls, silent start and sound preference', async ({ page }) => {
  await page.setViewportSize({width:390,height:844});
  await page.emulateMedia({reducedMotion:'reduce'});
  await page.goto('/abyss/rift');
  await expect(page.locator('#rift-start')).toBeEnabled();
  expect(await page.evaluate(() => window.RiftAudio.context)).toBeNull();
  await expect(page.locator('.rift-touch')).toBeVisible();
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
  await page.locator('#rift-start').click();
  const move = page.getByRole('button',{name:'Move right',exact:true});
  const box = await move.boundingBox();
  const initial = (await (await page.request.get('/api/abyss/rift')).json()).run.player.x;
  await page.mouse.move(box.x+box.width/2,box.y+box.height/2);
  await page.mouse.down();
  await expect.poll(async () => (await (await page.request.get('/api/abyss/rift')).json()).run.player.x).toBeGreaterThan(initial+20);
  await page.mouse.up();
  await page.locator('#rift-sound').click();
  await expect(page.locator('#rift-sound')).toHaveText('Sound off');
  await page.reload();
  await expect(page.locator('#rift-sound')).toHaveText('Sound off');
  await page.screenshot({path:'test-results/rift-mobile.png'});
});
