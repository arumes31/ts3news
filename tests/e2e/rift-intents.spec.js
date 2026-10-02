const {test,expect}=require('@playwright/test');
const read=async page=>(await(await page.request.get('/api/abyss/rift')).json()).run;

test('ability taps retain press order across an in-flight request and casting recovery',async({page})=>{
  await page.goto('/abyss/rift');await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();const run=await read(page),sent=[];
  let release,notify;const blocked=new Promise(resolve=>notify=resolve);
  await page.route('**/api/abyss/rift',async route=>{const body=route.request().postDataJSON();if(body?.kind==='step'){if(body.input.skill)sent.push(body.input.skill);if(notify){const ready=notify;notify=null;await new Promise(resolve=>{release=resolve;ready();});}}await route.continue();});
  await blocked;await page.keyboard.press('Digit2');await page.locator('#rift-signatures button').first().click();release();
  await expect.poll(()=>sent.slice(0,2)).toEqual([run.build.skills[1].id,run.build.signatures[0].id]);
  await expect.poll(async()=>(await read(page)).stats.skills_cast).toBeGreaterThanOrEqual(2);await page.keyboard.press('Escape');
});

test('cancelled pointer input discards its queued ability before the next request',async({page})=>{
  await page.goto('/abyss/rift');await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();let release,notify;const blocked=new Promise(resolve=>notify=resolve);
  await page.route('**/api/abyss/rift',async route=>{if(route.request().postDataJSON()?.kind==='step'&&notify){const ready=notify;notify=null;await new Promise(resolve=>{release=resolve;ready();});}await route.continue();});
  await blocked;const button=page.locator('#rift-signatures button').first();await button.hover();await page.mouse.down();await button.dispatchEvent('pointercancel',{pointerId:1});await page.mouse.up();release();
  await page.waitForTimeout(500);expect((await read(page)).stats.skills_cast).toBe(0);await page.keyboard.press('Escape');
});

test('tap-once ability mode persists and a held key requests only one cast',async({page})=>{
  const skills=[];page.on('request',request=>{if(request.method()==='POST'&&request.url().endsWith('/api/abyss/rift')){const body=request.postDataJSON();if(body.kind==='step'&&body.input.skill)skills.push(body.input.skill);}});
  await page.goto('/abyss/rift');await page.locator('.rift-settings > summary').click();await page.locator('#rift-ability-mode').selectOption('tap');await page.reload();await expect(page.locator('#rift-ability-mode')).toHaveValue('tap');await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();
  await page.keyboard.down('KeyQ');await expect.poll(()=>skills.length).toBe(1);await page.waitForTimeout(650);expect(skills.length).toBe(1);await page.keyboard.up('KeyQ');await page.keyboard.press('Escape');
  await page.locator('.rift-settings > summary').click();await page.locator('#rift-ability-mode').selectOption('hold');await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();await page.keyboard.down('KeyQ');await expect.poll(()=>skills.length).toBeGreaterThan(2);await page.keyboard.up('KeyQ');await page.keyboard.press('Escape');
});

test('expired and paused ability queues cannot fire after recovery',async({page})=>{
  await page.goto('/abyss/rift');await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();await page.keyboard.press('Escape');const run=await read(page);
  const result=await page.evaluate(async run=>{
    const intent=window.RiftIntents;run.player.cooldown=0;run.player.mana=100;run.skill_timers={};intent.sync(run,true);intent.press('signature0');await new Promise(resolve=>setTimeout(resolve,1250));const expired=intent.take(run,()=>false,false);
    intent.press('signature0');intent.reset();const paused=intent.take(run,()=>false,false);intent.press('signature0');intent.sync(run,true);const replay=intent.take(run,()=>false,false);return {expired,paused,replay};
  },run);
  for(const value of Object.values(result))expect(value.skill).toBe('');
});

test('movement and basic attacks remain usable while a queued ability is unavailable and modifiers do not enqueue actions',async({page})=>{
  const inputs=[];page.on('request',request=>{if(request.method()==='POST'&&request.url().endsWith('/api/abyss/rift')){const body=request.postDataJSON();if(body.kind==='step')inputs.push(body.input);}});
  await page.goto('/abyss/rift');await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();await page.keyboard.press('KeyQ');await expect.poll(async()=>(await read(page)).stats.skills_cast).toBe(1);
  await page.keyboard.press('KeyQ');await page.keyboard.down('KeyD');await page.keyboard.down('KeyJ');await expect.poll(()=>inputs.some(i=>i.x===1&&i.attack)).toBe(true);await expect.poll(async()=>(await read(page)).stats.attacks).toBeGreaterThan(0);await page.keyboard.up('KeyD');await page.keyboard.up('KeyJ');await page.keyboard.press('Escape');
  await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();const before=(await read(page)).stats.skills_cast;
  await page.locator('#rift-canvas').evaluate(canvas=>{for(const modifier of ['ctrlKey','altKey','metaKey'])canvas.dispatchEvent(new KeyboardEvent('keydown',{code:'Digit3',[modifier]:true,bubbles:true}));});
  await page.waitForTimeout(400);expect((await read(page)).stats.skills_cast).toBe(before);await page.keyboard.press('Digit3');await expect.poll(async()=>(await read(page)).stats.skills_cast).toBeGreaterThan(before);await page.keyboard.press('Escape');
});

test('opposing keyboard directions follow the newest press and restore the held direction',async({page})=>{
  const inputs=[];page.on('request',request=>{if(request.method()==='POST'&&request.url().endsWith('/api/abyss/rift')){const body=request.postDataJSON();if(body.kind==='step')inputs.push(body.input);}});
  await page.goto('/abyss/rift');await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();await page.keyboard.down('KeyA');await expect.poll(()=>inputs.at(-1)?.x).toBe(-1);
  await page.keyboard.down('KeyD');await expect.poll(()=>inputs.at(-1)?.x).toBe(1);await page.keyboard.up('KeyD');await expect.poll(()=>inputs.at(-1)?.x).toBe(-1);await page.keyboard.up('KeyA');await page.keyboard.press('Escape');
});

test('ability queue stays bounded and expired entries do not reject a fresh press',async({page})=>{
  await page.goto('/abyss/rift');await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();await page.keyboard.press('Escape');const run=await read(page);
  const result=await page.evaluate(async run=>{
    const intent=window.RiftIntents;run.player.cooldown=0;run.player.mana=100;run.skill_timers={};intent.sync(run,true);
    for(let i=0;i<9;i++)intent.press('signature0');const bounded=Array.from({length:9},()=>intent.take(run,()=>false,false).skill).filter(Boolean).length;
    for(let i=0;i<8;i++)intent.press('signature0');await new Promise(resolve=>setTimeout(resolve,1250));intent.press('ultimate');return {bounded,fresh:intent.take(run,()=>false,false).skill};
  },run);
  expect(result).toEqual({bounded:8,fresh:run.build.ultimate.id});
});
