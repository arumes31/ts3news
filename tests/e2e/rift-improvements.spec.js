const { test, expect } = require('@playwright/test');
const path = require('node:path');

test('pausing cancels a queued checkpoint before rewards are banked',async({page})=>{
  await page.goto('/abyss/rift?scenario=checkpoint');await page.locator('#rift-auto').uncheck();
  let release,notify;const pending=new Promise(resolve=>notify=resolve),exits=[];
  await page.route('**/api/abyss/rift',async route=>{
    const kind=route.request().postDataJSON()?.kind;if(kind==='exit')exits.push(kind);
    if(kind==='step'&&notify){const ready=notify;notify=null;await new Promise(resolve=>{release=resolve;ready();});}
    await route.continue();
  });
  await page.locator('#rift-start').click();await pending;
  try{await page.locator('#rift-exit').click();await page.keyboard.press('Escape');}finally{release();}
  await expect(page.locator('#rift-overlay-title')).toHaveText('A moment by the lantern.');
  await expect(page.locator('#rift-banked')).toHaveText('0 gold · 0 items');expect(exits).toEqual([]);
});

test('checkpoint clicks queue once behind an in-flight combat update',async({page})=>{
  await page.goto('/abyss/rift?scenario=checkpoint');await page.locator('#rift-auto').uncheck();
  let release,notify;const pending=new Promise(resolve=>notify=resolve),exits=[];
  await page.route('**/api/abyss/rift',async route=>{
    const kind=route.request().postDataJSON()?.kind;
    if(kind==='exit')exits.push(kind);
    if(kind==='step'&&notify){const ready=notify;notify=null;await new Promise(resolve=>{release=resolve;ready();});}
    await route.continue();
  });
  await page.locator('#rift-start').click();await pending;
  try{await expect(page.locator('#rift-exit')).toBeEnabled();await page.locator('#rift-exit').click();await page.locator('#rift-exit').evaluate(button=>button.click());}finally{release();}
  await expect(page.locator('#rift-banked')).toHaveText('30 gold · 1 item');expect(exits).toHaveLength(1);
});

test('loot inspection and the bank receipt match confirmed inventory delivery',async({page})=>{
  await page.goto('/abyss/rift?scenario=checkpoint');await page.locator('#rift-auto').uncheck();
  const data=await(await page.request.get('/api/abyss/rift')).json(),gear=data.run.drops[0].gear;
  await expect(page.locator('#rift-loot-count')).toHaveText('1 item pending');
  await page.locator('#rift-loot details > summary').click();
  await expect(page.locator('#rift-loot')).toContainText(gear.Name);
  await expect(page.locator('#rift-loot')).toContainText('Maximum durability');
  await expect(page.locator('#rift-loot')).toContainText('Mission 1 · Tier 1');
  await expect(page.locator('#rift-loot summary')).toHaveAttribute('title','Found: '+gear.found_boss);
  await page.locator('#rift-loot-sort').selectOption('slot');
  await page.locator('#rift-start').click();await expect(page.locator('#rift-checkpoint-total')).toHaveText('30 gold · 1 item ready to bank');await page.locator('#rift-exit').click();
  await expect(page.locator('#rift-banked')).toHaveText('30 gold · 1 item');
  await expect(page.locator('#rift-loot-count')).toHaveText('0 items pending');
  await page.locator('#rift-receipt > summary').click();
  await expect(page.locator('#rift-receipt-list')).toContainText(gear.Name);
  await expect(page.locator('#rift-receipt a')).toHaveAttribute('href','/inventory');
  await page.locator('#rift-receipt-search').fill('absent item');
  await expect(page.locator('#rift-receipt-empty')).toBeVisible();
  await page.locator('#rift-receipt-search').fill('');
  await page.evaluate(()=>{navigator.clipboard.writeText=async text=>window.copiedReceipt=text;});
  await page.locator('#rift-copy-receipt').click();
  const copied=await page.evaluate(()=>window.copiedReceipt);expect(copied).toContain(gear.Name);expect(copied).toContain('30 gold');expect(copied).not.toContain(data.run.id);
  await page.goto('/abyss/rift');await page.locator('#rift-receipt > summary').click();
  await expect(page.locator('#rift-receipt-list')).toContainText(gear.Name);
});

for(const corruption of ['missing run','invalid health','future schema','unknown status'])test('invalid response recovery preserves confirmed rewards: '+corruption,async({page})=>{
  await page.goto('/abyss/rift');await page.locator('#rift-start').click();
  const before=await page.locator('#rift-banked').textContent();
  let corrupted=false;
  await page.route('**/api/abyss/rift',async route=>{
    if(route.request().postDataJSON()?.kind==='step'&&!corrupted){
      corrupted=true;const response=await route.fetch(),data=await response.json();
      data.run.banked_gold=9999999;
      if(corruption==='missing run')delete data.run;
      if(corruption==='invalid health')data.run.player.hp='broken';
      if(corruption==='future schema')data.run.schema=999;
      if(corruption==='unknown status')data.run.status='future';
      await route.fulfill({response,json:data});return;
    }
    await route.continue();
  });
  await expect(page.locator('#rift-start')).toHaveText('Recover expedition');
  await expect(page.locator('#rift-banked')).toHaveText(before);
  await page.locator('#rift-start').click();await expect(page.locator('#rift-start')).toHaveText('Resume expedition');
  await expect(page.locator('#rift-banked')).not.toContainText('9999999');
});

test('display presets preview explicit changes, persist custom values and reset safely',async({page})=>{
  await page.goto('/abyss/rift');
  await page.locator('.rift-settings > summary').click();
  await page.locator('#rift-display-preset').selectOption('accessible');
  await expect(page.locator('#rift-preset-description')).toContainText('Larger text');
  await expect(page.locator('#rift-text-scale')).toHaveValue('1');
  await page.locator('#rift-apply-preset').click();
  await expect(page.locator('#rift-text-scale')).toHaveValue('1.25');
  await expect(page.locator('#rift-hazard-contrast')).toBeChecked();
  await page.locator('#rift-enemy-names').selectOption('boss');
  await page.locator('#rift-damage-numbers').uncheck();
  await page.reload();await page.locator('.rift-settings > summary').click();
  await expect(page.locator('#rift-enemy-names')).toHaveValue('boss');
  await expect(page.locator('#rift-damage-numbers')).not.toBeChecked();
  await expect(page.locator('#rift-text-scale')).toHaveValue('1.25');
  await page.setViewportSize({width:390,height:844});
  expect(await page.evaluate(()=>document.documentElement.scrollWidth<=window.innerWidth)).toBe(true);
  await page.locator('#rift-reset-display').click();
  await expect(page.locator('#rift-enemy-names')).toHaveValue('all');
  await expect(page.locator('#rift-damage-numbers')).toBeChecked();
  await expect(page.locator('#rift-text-scale')).toHaveValue('1');
});

test('corrupt display values fall back individually and low-power rendering limits frames',async({page})=>{
  await page.addInitScript(()=>localStorage.setItem('riftDisplay',JSON.stringify({version:1,enemyNames:'boss',healthBars:'no',fps:0,textScale:1000,particles:false,damageNumbers:false,unknown:true})));
  await page.goto('/abyss/rift');await page.locator('.rift-settings > summary').click();
  await expect(page.locator('#rift-enemy-names')).toHaveValue('boss');
  await expect(page.locator('#rift-enemy-health')).toBeChecked();
  await expect(page.locator('#rift-text-scale')).toHaveValue('1');
  await expect(page.locator('#rift-background-particles')).not.toBeChecked();
  await page.locator('#rift-render-rate').selectOption('30');
  const before=await page.evaluate(()=>window.RiftRenderer.frameCount);await page.waitForTimeout(1000);
  const frames=await page.evaluate(()=>window.RiftRenderer.frameCount);
  expect(frames-before).toBeGreaterThan(5);expect(frames-before).toBeLessThanOrEqual(33);
});

test('bestiary filters compose against the live roster and clear together',async({page})=>{
  await page.goto('/abyss/rift');
  const catalog=(await(await page.request.get('/api/abyss/rift')).json()).bestiary;
  await page.locator('.rift-bestiary > summary').click();
  const monster=catalog.find(unit=>unit.kind==='boss');
  await page.locator('#rift-monster-tier').selectOption(monster.tier);
  await page.locator('#rift-monster-element').selectOption(monster.element||'physical');
  await page.locator('#rift-monster-style').selectOption('Area attacks');
  const expected=catalog.filter(unit=>unit.kind==='boss'&&unit.tier===monster.tier&&(unit.element||'physical')===(monster.element||'physical'));
  await expect(page.locator('#rift-monsters article:visible')).toHaveCount(expected.length);
  await expect(page.locator('#rift-monster-matches')).toHaveText(expected.length+' of '+catalog.length+' monsters');
  await page.locator('#rift-monster-search').fill('not an abyss monster');
  await expect(page.locator('#rift-monsters-empty')).toBeVisible();
  await page.locator('#rift-monster-clear').click();
  await expect(page.locator('#rift-monsters article:visible')).toHaveCount(catalog.length);
  await expect(page.locator('#rift-monster-search')).toBeFocused();
});

test('monster inspection is keyboard accessible and previews respect reduced motion',async({page})=>{
  await page.goto('/abyss/rift');
  const catalog=(await(await page.request.get('/api/abyss/rift')).json()).bestiary;
  const monster=catalog.find(unit=>unit.kind==='boss');
  await page.locator('.rift-bestiary > summary').click();
  await page.locator('#rift-monster-search').fill(monster.name);
  const inspect=page.getByRole('button',{name:'Inspect '+monster.name,exact:true});
  await inspect.focus();await page.keyboard.press('Enter');
  await expect(page.locator('#rift-monster-title')).toHaveText(monster.name);
  await expect(page.locator('#rift-monster-title')).toBeFocused();
  const health=page.locator('#rift-monster-stats > div').filter({has:page.getByText('Health',{exact:true})}).locator('dd');
  await expect(health).toHaveText(new Intl.NumberFormat('en-US',{maximumFractionDigits:1}).format(monster.max_hp));
  await expect(page.locator('#rift-monster-tip')).toContainText('jump or move clear');
  await page.locator('#rift-monster-pose').selectOption('attack');
  const position=()=>page.locator('#rift-monster-preview').evaluate(node=>node.style.backgroundPosition);
  const first=await position();await expect.poll(position).not.toBe(first);
  await page.locator('.rift-settings > summary').click();
  await page.locator('#rift-reduced').check();
  const still=await position();await page.waitForTimeout(400);expect(await position()).toBe(still);
  await page.locator('#rift-monster-pose').selectOption('defeat');
  const asset=await page.locator('#rift-monster-preview').evaluate(node=>node.style.backgroundImage);
  expect(asset).toContain('/static/abyss_');
  await page.locator('#rift-monster-pose').focus();await page.keyboard.press('Escape');await expect(page.locator('#rift-monster-detail')).toBeHidden();await expect(inspect).toBeFocused();
});

test.beforeEach(async({page})=>{
  // Fast frontend iteration against the unchanged production simulation.
  // Final release verification runs without this override.
  if(process.env.RIFT_WORKSPACE_ASSETS)await page.route(/\/static\/rift(?:_audio)?\.js(?:\?|$)/,route=>route.fulfill({path:path.resolve(__dirname,'../../internal/bot/webassets',new URL(route.request().url()).pathname.split('/').pop()),contentType:'application/javascript'}));
});

test('brief action taps survive an in-flight movement request',async({page})=>{
  await page.goto('/abyss/rift?subclass=vanguard');
  await page.locator('#rift-start').click();
  let release, intercept;
  const pending=new Promise(resolve=>intercept=resolve);
  await page.route('**/api/abyss/rift',async route=>{
    const body=route.request().postDataJSON();
    if(body?.kind==='step'&&intercept){const notify=intercept;intercept=null;await new Promise(resolve=>{release=resolve;notify();});}
    await route.continue();
  });
  await pending;
  await page.keyboard.press('Space');
  release();
  await expect.poll(async()=>(await(await page.request.get('/api/abyss/rift')).json()).run.skill_timers.jump||0).toBeGreaterThan(0);
  await page.keyboard.press('Escape');
});

test('repeated start clicks create only one expedition while audio resumes',async({page})=>{
  await page.goto('/abyss/rift');
  await expect(page.locator('#rift-start')).toBeEnabled();
  const starts=[];page.on('request',request=>{if(request.method()==='POST'&&['start','resume'].includes(request.postDataJSON()?.kind))starts.push(request);});
  await page.evaluate(()=>{
    const original=window.RiftAudio.setActive;
    let calls=0;
    window.RiftAudio.setActive=async(...args)=>{const delay=++calls===1?50:450;await new Promise(resolve=>setTimeout(resolve,delay));return original(...args);};
    document.querySelector('#rift-start').click();
    document.querySelector('#rift-start').click();
  });
  await expect(page.locator('#rift-overlay')).toBeHidden();
  await page.waitForTimeout(600);
  expect(starts).toHaveLength(1);
  await page.keyboard.press('Escape');
});

test('audio failure does not prevent an expedition from starting',async({page})=>{
  await page.goto('/abyss/rift');
  await page.evaluate(()=>{window.RiftAudio.setActive=async()=>{throw new Error('Audio unavailable');};});
  await page.locator('#rift-start').click();
  await expect(page.locator('#rift-overlay')).toBeHidden();
  await page.keyboard.press('Escape');
  await expect(page.locator('#rift-overlay-title')).toHaveText('A moment by the lantern.');
});

test('Escape pauses when a settings input has focus',async({page})=>{
  await page.goto('/abyss/rift');
  await page.locator('#rift-start').click();
  await page.locator('#rift-campaign > summary').click();
  await page.locator('#rift-auto').focus();
  await page.keyboard.press('Escape');
  await expect(page.locator('#rift-overlay-title')).toHaveText('A moment by the lantern.');
});

test('pause cannot overwrite a terminal result received by a pending step',async({page})=>{
  await page.goto('/abyss/rift');
  await page.locator('#rift-start').click();
  let release, intercepted;
  const pending=new Promise(resolve=>intercepted=resolve);
  await page.route('**/api/abyss/rift',async route=>{
    if(route.request().postDataJSON()?.kind==='step'&&intercepted){
      const response=await route.fetch(),data=await response.json();
      data.run.status='defeated';data.run.player.hp=0;
      const notify=intercepted;intercepted=null;
      await new Promise(resolve=>{release=resolve;notify();});
      await route.fulfill({response,json:data});return;
    }
    await route.continue();
  });
  await pending;await page.keyboard.press('Escape');release();
  await expect(page.locator('#rift-overlay-title')).toHaveText('The rift takes its toll.');
  await page.waitForTimeout(300);
  await expect(page.locator('#rift-overlay-title')).toHaveText('The rift takes its toll.');
});

test('campaign search combines with difficulty, progress and favorites',async({page})=>{
  await page.goto('/abyss/rift');
  await expect(page.locator('[data-level]')).toHaveCount(100);
  await page.locator('#rift-region').selectOption('9');
  await page.locator('#rift-difficulty').selectOption('Mythic');
  await page.locator('#rift-completion').selectOption('unfinished');
  await expect(page.locator('[data-level]:visible')).toHaveCount(10);
  await page.locator('#rift-mission-search').fill('100');
  await expect(page.locator('[data-level]:visible')).toHaveCount(1);
  await page.locator('[data-level="100"]').click();
  await page.locator('#rift-favorite').click();
  await page.locator('#rift-favorites-only').check();
  await page.locator('#rift-compact').check();
  await page.reload();
  await expect(page.locator('[data-level]:visible')).toHaveCount(1);
  await expect(page.locator('#rift-level-description')).toContainText('Mission 100');
  await expect(page.locator('#rift-start')).toHaveText('Enter mission 100');
  await expect(page.locator('#rift-favorite')).toHaveAttribute('aria-pressed','true');
  await expect(page.locator('#rift-levels')).toHaveClass(/compact/);
  await page.locator('#rift-mission-search').fill('not a mission');
  await expect(page.locator('#rift-filter-empty')).toBeVisible();
  await page.locator('#rift-clear-filters').click();
  await expect(page.locator('[data-level]:visible')).toHaveCount(100);
});

test('mission shortcuts do not replace an active expedition',async({page})=>{
  await page.goto('/abyss/rift');
  await page.locator('#rift-next-mission').click();
  await expect(page.locator('#rift-level-description')).toContainText('Mission 2');
  await page.locator('#rift-previous-mission').click();
  await expect(page.locator('#rift-level-description')).toContainText('Mission 1');
  await page.locator('#rift-unfinished').click();
  await expect(page.locator('#rift-level-description')).toContainText('Mission 2');
  await page.locator('#rift-start').click();
  await page.locator('#rift-campaign > summary').click();
  await expect(page.locator('#rift-next-mission')).toBeDisabled();
  await expect(page.locator('#rift-previous-mission')).toBeDisabled();
  await expect(page.locator('#rift-unfinished')).toBeDisabled();
  await page.keyboard.press('Escape');
});

test('completion filters reflect confirmed regional progress',async({page})=>{
  await page.goto('/abyss/rift?scenario=checkpoint&room=final');
  await page.locator('#rift-start').click();
  await expect.poll(async()=>(await(await page.request.get('/api/abyss/rift')).json()).run.level.id).toBe(2);
  await page.keyboard.press('Escape');
  await page.locator('#rift-campaign > summary').click();
  await page.locator('#rift-completion').selectOption('complete');
  await expect(page.locator('[data-level]:visible')).toHaveCount(1);
  await expect(page.locator('#rift-filter-count')).toHaveText('1 of 100 missions');
  await expect(page.locator('#rift-region-progress')).toContainText('Mossbound Ruins: 1/10');
  await page.locator('#rift-show-selected').click();
  await expect(page.locator('[data-level="2"]')).toBeVisible();
});

test('combat feedback uses confirmed stats and exposes health meters',async({page})=>{
  await page.goto('/abyss/rift?subclass=geomancer');
  await page.locator('#rift-start').click();
  await expect(page.locator('#rift-enemy-count')).toContainText('remaining');
  await expect(page.getByRole('meter',{name:'Player health',exact:true})).toHaveAttribute('aria-valuemax','340');
  await expect(page.getByRole('meter',{name:'Player mana',exact:true})).toHaveAttribute('aria-valuemax','100');
  await page.keyboard.press('q');
  await expect.poll(async()=>(await(await page.request.get('/api/abyss/rift')).json()).run.stats.skills_cast).toBeGreaterThan(0);
  await expect(page.locator('#rift-barrier-state')).not.toHaveText('No barrier');
  await page.keyboard.press('Escape');
  const before=(await(await page.request.get('/api/abyss/rift')).json()).run.stats.seconds;
  await page.waitForTimeout(300);
  expect((await(await page.request.get('/api/abyss/rift')).json()).run.stats.seconds).toBe(before);
  await page.locator('.rift-run-statistics > summary').click();
  await expect(page.locator('#rift-statistics')).toContainText('Mana spent');
  await expect(page.locator('#rift-announcer')).toHaveText('Expedition paused.');
});

test('reduced motion persists and the expanded mobile controls do not overflow',async({page})=>{
  await page.setViewportSize({width:390,height:844});
  await page.goto('/abyss/rift');
  await page.locator('.rift-settings > summary').click();
  await page.locator('#rift-reduced').check();
  await page.reload();
  await page.locator('.rift-settings > summary').click();
  await expect(page.locator('#rift-reduced')).toBeChecked();
  expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
});

test('keyboard activation of action buttons performs the labeled action',async({page})=>{
  await page.goto('/abyss/rift');await page.locator('#rift-start').click();
  const attack=page.locator('[data-hold="attack"]');await attack.focus();await page.keyboard.press('Enter');
  await expect.poll(async()=>(await(await page.request.get('/api/abyss/rift')).json()).run.stats.attacks).toBeGreaterThan(0);
  await expect.poll(async()=>(await(await page.request.get('/api/abyss/rift')).json()).run.player.cooldown).toBe(0);
  await page.keyboard.press('Space');
  await expect.poll(async()=>(await(await page.request.get('/api/abyss/rift')).json()).run.stats.attacks).toBeGreaterThan(1);
  expect((await(await page.request.get('/api/abyss/rift')).json()).run.stats.jumps).toBe(0);
  await page.keyboard.press('Escape');
});

test('initial API failure has a working retry action',async({page})=>{
  let fail=true;
  await page.route('**/api/abyss/rift',route=>fail&&route.request().method()==='GET'?route.fulfill({status:503,body:'Temporarily unavailable'}):route.continue());
  await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toHaveText('Retry loading');
  fail=false;await page.locator('#rift-start').click();await expect(page.locator('#rift-start')).toHaveText('Enter the ruins →');
  await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();await page.keyboard.press('Escape');
});

test('corrupt audio levels use finite defaults and closed audio can reopen',async({page})=>{
  await page.addInitScript(()=>{localStorage.setItem('riftAudio:effects','"invalid"');localStorage.setItem('riftAudio:ambience','999');});
  await page.goto('/abyss/rift');
  expect(await page.evaluate(()=>window.RiftAudio.effects)).toBe(.65);
  expect(await page.evaluate(()=>window.RiftAudio.ambience)).toBe(1);
  await page.locator('#rift-start').click();await page.keyboard.press('Escape');
  await page.evaluate(()=>window.RiftAudio.context.close());
  await page.locator('#rift-start').click();
  await expect.poll(()=>page.evaluate(()=>window.RiftAudio.context.state)).toBe('running');
  await page.keyboard.press('Escape');
});

test('failed artwork can be reloaded from the start panel',async({page})=>{
  let fail=true;
  await page.route('**/static/rift_regions.png*',route=>fail?route.abort():route.continue());
  await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toHaveText('Reload artwork');
  fail=false;await page.locator('#rift-start').click();await expect(page.locator('#rift-start')).toHaveText('Enter the ruins →');
});

test('restored browser history reloads the confirmed expedition without autoplay',async({page})=>{
  await page.goto('/abyss/rift');await page.locator('#rift-start').click();await page.keyboard.press('Escape');
  await expect(page.locator('#rift-overlay-title')).toHaveText('A moment by the lantern.');
  await page.evaluate(()=>window.dispatchEvent(new PageTransitionEvent('pageshow',{persisted:true})));
  await expect(page.locator('#rift-overlay-title')).toHaveText('Your expedition awaits.');
  await expect(page.locator('#rift-start')).toHaveText('Resume expedition');
});
