const { test, expect } = require('@playwright/test');

test('bestiary mirrors the live Abyss roster and WASD/Space control combat',async({page})=>{
  const errors=[];page.on('pageerror',e=>errors.push(e.message));
  await page.goto('/abyss/rift?subclass=elementalist');
  const catalog=(await(await page.request.get('/api/abyss/rift')).json()).bestiary;
  expect(catalog.length).toBeGreaterThan(110);
  await expect(page.locator('#rift-monster-count')).toHaveText(catalog.length+' monsters');
  await page.locator('.rift-bestiary > summary').click();
  await expect(page.locator('#rift-monsters article')).toHaveCount(catalog.length);
  const rows=await page.locator('#rift-monsters article').evaluateAll(nodes=>nodes.map(n=>n.dataset.artKey));
  expect(rows.sort()).toEqual(catalog.map(m=>m.art_key).sort());
  const frames=await page.evaluate(list=>list.map(m=>{const f=window.RiftBestiary.frame(m,'attack',1);return {name:m.name,source:f.source,rig:f.rig};}),catalog);
  for(const frame of frames){expect(frame.source,frame.name).toBeTruthy();expect(frame.source.width).toBeGreaterThan(0);}
  await page.locator('#rift-monster-search').fill('Chronos');
  await expect(page.locator('#rift-monsters article:visible')).toHaveCount(1);
  await page.screenshot({path:'test-results/rift-bestiary.png',fullPage:true});
  await page.locator('#rift-start').click();
  await expect(page.locator('#rift-overlay')).toBeHidden();
  const read=async()=>(await(await page.request.get('/api/abyss/rift')).json()).run;
  let run=await read();
  for(const [key,axis,direction] of [['d','x',1],['a','x',-1],['w','y',-1],['s','y',1]]){
    const before=run.player[axis];await page.keyboard.down(key);
    await expect.poll(async()=>((await read()).player[axis]-before)*direction).toBeGreaterThan(12);
    await page.keyboard.up(key);run=await read();
  }
  const scroll=await page.evaluate(()=>scrollY);
  await page.keyboard.down('Space');
  await expect.poll(async()=>(await read()).player.jump).toBeGreaterThan(0);
  await page.keyboard.up('Space');expect(await page.evaluate(()=>scrollY)).toBe(scroll);
  run=await read();const ids=new Set(catalog.map(m=>m.art_key));
  for(const room of run.encounter_plan)for(const mob of room)expect(ids.has(mob.art_key)).toBe(true);
  await page.keyboard.press('Escape');expect(errors).toEqual([]);
});

test('empty regular loadout still supports class combat', async ({ page }) => {
  await page.goto('/abyss/rift?subclass=geomancer');
  await expect(page.locator('#rift-start')).toBeEnabled();
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
    await expect(page.locator('#rift-start')).toBeEnabled();
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

test('clear all three rooms, defeat the catalog boss and bank the expedition', async ({ page }) => {
  test.setTimeout(180_000);
  const errors=[];page.on('pageerror',error=>errors.push(error.message));
  await page.goto('/abyss/rift?subclass=bloodblade');
  await page.locator('#rift-auto').uncheck();
  await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();
  const held=new Set();
  async function controls(wanted){for(const key of [...held])if(!wanted.has(key)){await page.keyboard.up(key);held.delete(key);}for(const key of wanted)if(!held.has(key)){await page.keyboard.down(key);held.add(key);}}
  const started=Date.now();let complete=false,expectedFightGold=0,expectedItems=0;
  while(Date.now()-started<145_000){
    const run=(await(await page.request.get('/api/abyss/rift')).json()).run;
    expect(run.status,'expedition should remain survivable').not.toBe('defeated');
    if(run.status==='complete'){complete=true;expect(expectedFightGold).toBeGreaterThan(0);expect(expectedItems).toBeGreaterThan(0);expect(run.banked_gold).toBe(expectedFightGold+(run.banked_objective_gold||0));expect(run.banked_items.length).toBe(expectedItems);for(const drop of run.drops.filter(d=>d.gear))expect(drop.gear.found_boss).toContain(run.level.name);break;}
    if(run.status==='cleared'){
      const unbanked=run.drops.filter(drop=>!drop.banked);expectedFightGold+=unbanked.reduce((sum,drop)=>sum+drop.gold,0);expectedItems+=unbanked.filter(drop=>drop.gear).length;
      await controls(new Set());
      await expect(page.locator('#rift-next')).toBeVisible();
      if(run.room===2)await page.screenshot({path:'test-results/rift-boss-cleared.png'});
      await page.locator('#rift-next').click();
      await expect(page.locator('#rift-next')).toBeHidden();continue;
    }
    const p=run.player,target=run.enemies.filter(e=>e.hp>0).sort((a,b)=>Math.abs(a.x-p.x)-Math.abs(b.x-p.x))[0];
    const dx=target.x-p.x,dy=target.y-p.y,wanted=new Set(['Space']);
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
  await page.locator('#rift-auto').uncheck();
  await expect(page.locator('#rift-start')).toHaveText('Resume expedition');
  await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();
  await expect(page.locator('#rift-loot')).not.toContainText('Your next discovery');
  const before = (await (await page.request.get('/api/abyss/rift')).json()).run;
  expect(before.drops[0].gear.ID).toMatch(/^ABYSS_/);
  await page.locator('#rift-exit').click();
  await expect(page.locator('#rift-banked')).toHaveText('30 gold · 1 item');
  const banked = (await (await page.request.get('/api/abyss/rift')).json()).run;
  await page.request.post('/api/abyss/rift', {data:{kind:'exit',run_id:banked.id,revision:banked.revision,request_id:'retry-bank-request',input:{}}});
  await page.goto('/abyss/rift');
  await expect(page.locator('#rift-banked')).toHaveText('30 gold · 1 item');
});

test('100 missions are selectable and the final region survives reload',async({page})=>{
  const errors=[];page.on('pageerror',error=>errors.push(error.message));
  await page.goto('/abyss/rift');
  await expect(page.locator('[data-level]')).toHaveCount(100);
  const data=await(await page.request.get('/api/abyss/rift')).json();
  expect(new Set(data.levels.map(l=>l.name)).size).toBe(100);
  await page.locator('#rift-region').selectOption('9');
  await expect(page.locator('[data-level]:visible')).toHaveCount(10);
  await page.locator('[data-level="100"]').click();
  await expect(page.locator('#rift-level-description')).toContainText('Obsidian Citadel');
  await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();
  const before=await(await page.request.get('/api/abyss/rift')).json();
  expect(before.run.level.id).toBe(100);expect(before.run.level.rooms[0].obstacles.length).toBeGreaterThan(2);
  await page.keyboard.press('Escape');
  await page.screenshot({path:'test-results/rift-citadel.png'});
  await page.reload();
  await expect(page.locator('#rift-start')).toHaveText('Resume expedition');
  expect((await(await page.request.get('/api/abyss/rift')).json()).run.level.id).toBe(100);
  expect(errors).toEqual([]);
});

test('seamless tiers bank once without navigation and pause stops the transition',async({page})=>{
  await page.goto('/abyss/rift?scenario=checkpoint');
  await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();
  await page.keyboard.press('Escape');
  await expect(page.locator('#rift-overlay-title')).toHaveText('A moment by the lantern.');
  await page.waitForTimeout(1400);
  const read=async()=>(await(await page.request.get('/api/abyss/rift')).json()).run;
  expect((await read()).room).toBe(0);
  let navigations=0;page.on('framenavigated',frame=>{if(frame===page.mainFrame())navigations++;});
  const assets=[];page.on('request',request=>{if(request.resourceType()==='image'||/\/rift_creature_[^/]+\.png/.test(request.url()))assets.push(request.url());});
  await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();
  await expect.poll(async()=>(await read()).room).toBe(1);
  const next=await read();expect(next.banked_gold).toBe(30);expect(next.banked_items).toHaveLength(1);
  expect(navigations).toBe(0);expect(assets).toEqual([]);
  await expect(page.locator('#rift-overlay')).toBeHidden();
  await page.keyboard.press('Escape');
});

test('seamless boss clearance starts the next mission and records completion',async({page})=>{
  await page.goto('/abyss/rift?scenario=checkpoint&room=final');
  await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();
  const read=async()=>(await(await page.request.get('/api/abyss/rift')).json()).run;
  await expect.poll(async()=>(await read()).level.id).toBe(2);
  await page.keyboard.press('Escape');
  const next=await read();expect(next.room).toBe(0);expect(next.completed_levels).toEqual([1]);expect(next.banked_gold).toBe(70);expect(next.banked_objective_gold).toBe(40);
  await page.goto('/abyss/rift');
  await expect(page.locator('#rift-progress')).toContainText('1/100 completed');
  await expect(page.locator('[data-level="1"]')).toHaveClass(/completed/);
  expect((await read()).level.id).toBe(2);
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
  await expect(page.locator('#rift-overlay')).toBeHidden();
  await expect(page.locator('#rift-pause')).toBeEnabled();
  const move = page.getByRole('button',{name:'Move right',exact:true});
  await move.scrollIntoViewIfNeeded();
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
