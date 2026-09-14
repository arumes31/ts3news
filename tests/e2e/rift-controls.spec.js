const { test, expect } = require('@playwright/test');

test('mouse attack and guard bindings apply only to the battlefield and persist',async({page})=>{
  await page.goto('/abyss/rift');await page.locator('#rift-controls-open').click();
  await page.locator('#rift-mouse-attack').selectOption('0');await page.locator('#rift-mouse-guard').selectOption('0');
  await expect(page.locator('#rift-binding-status')).toContainText('already assigned');await expect(page.locator('#rift-mouse-guard')).toHaveValue('-1');
  await page.locator('#rift-mouse-guard').selectOption('2');await page.locator('#rift-controls-close').click();await page.reload();
  await page.locator('#rift-controls-open').click();await expect(page.locator('#rift-mouse-guard')).toHaveValue('2');await page.locator('#rift-controls-close').click();
  await page.locator('#rift-start').click();const canvas=page.locator('#rift-canvas'),read=async()=>(await(await page.request.get('/api/abyss/rift')).json()).run;
  await canvas.click({position:{x:300,y:300}});await expect.poll(async()=>(await read()).stats.attacks).toBeGreaterThan(0);
  await canvas.hover({position:{x:300,y:300}});await page.mouse.down({button:'right'});
  await expect.poll(async()=>(await read()).player.guard).toBe(true);await page.mouse.up({button:'right'});await expect.poll(async()=>(await read()).player.guard).toBe(false);
  await page.keyboard.press('Escape');
});

test('toggle guard survives release and resets on pause for keyboard and touch actions',async({page})=>{
  await page.goto('/abyss/rift');await page.locator('#rift-controls-open').click();await page.locator('#rift-toggle-guard').check();await page.locator('#rift-controls-close').click();
  await page.locator('#rift-start').click();const read=async()=>(await(await page.request.get('/api/abyss/rift')).json()).run;
  await page.keyboard.press('l');await expect.poll(async()=>(await read()).player.guard).toBe(true);await page.waitForTimeout(250);expect((await read()).player.guard).toBe(true);
  await expect(page.locator('[data-bind="guard"]')).toHaveAttribute('aria-pressed','true');
  await page.locator('[data-bind="guard"]').click();await expect.poll(async()=>(await read()).player.guard).toBe(false);
  await page.locator('[data-bind="guard"]').click();await expect.poll(async()=>(await read()).player.guard).toBe(true);
  await page.keyboard.press('Escape');await expect(page.locator('[data-bind="guard"]')).toHaveAttribute('aria-pressed','false');
  await page.locator('#rift-start').click();await expect.poll(async()=>(await read()).player.guard).toBe(false);await page.keyboard.press('Escape');
});

test('basic guard and the equipped Iron Guard skill have separate actions',async({page})=>{
  await page.goto('/abyss/rift');await page.locator('#rift-controls-open').click();await page.locator('#rift-toggle-guard').check();await page.locator('#rift-controls-close').click();
  const inputs=[];page.on('request',request=>{if(request.method()==='POST'){const data=request.postDataJSON();if(data?.input)inputs.push(data.input);}});
  await page.locator('#rift-start').click();await page.locator('[data-bind="guard"]').click();
  await expect.poll(()=>inputs.some(input=>input.guard)).toBe(true);expect(inputs.some(input=>input.skill==='guard')).toBe(false);
  await page.locator('[data-bind="guard"]').click();await expect(page.locator('[data-bind="guard"]')).toHaveAttribute('aria-pressed','false');
  await page.locator('#rift-skills [data-hold="guard"]').click();
  await expect.poll(()=>inputs.some(input=>input.skill==='guard')).toBe(true);
  await expect(page.locator('[data-bind="guard"]')).toHaveAttribute('aria-pressed','false');await page.keyboard.press('Escape');
});

test('opening controls during audio startup cancels the pending expedition',async({page})=>{
  await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();
  const starts=[];page.on('request',request=>{if(request.method()==='POST'&&request.postDataJSON()?.kind==='start')starts.push(request);});
  await page.evaluate(()=>{window.RiftAudio.setActive=active=>active?new Promise(resolve=>window.releaseStartAudio=resolve):Promise.resolve();});
  await page.locator('#rift-start').click();await page.locator('#rift-controls-open').click();
  await page.evaluate(()=>window.releaseStartAudio());await page.waitForTimeout(250);
  await expect(page.locator('#rift-controls-dialog')).toBeVisible();expect(starts).toHaveLength(0);
});

test('remapped combat keys update prompts, persist and drive confirmed actions',async({page})=>{
  await page.goto('/abyss/rift');
  await page.locator('#rift-controls-open').click();
  await page.locator('[data-remap="attack"]').click();await page.keyboard.press('f');
  await expect(page.locator('[data-remap="attack"]')).toContainText('F');
  await page.locator('[data-remap="right"]').click();await page.keyboard.press('h');
  await page.locator('#rift-controls-close').click();
  await expect(page.locator('#rift-controls-open')).toBeFocused();
  await page.reload();await page.locator('#rift-start').click();
  await expect(page.locator('[data-hold="attack"] kbd')).toHaveText('F');
  const read=async()=>(await(await page.request.get('/api/abyss/rift')).json()).run;
  let before=await read();await page.keyboard.down('h');
  await expect.poll(async()=>(await read()).player.x-before.player.x).toBeGreaterThan(10);await page.keyboard.up('h');
  await page.keyboard.press('f');await expect.poll(async()=>(await read()).stats.attacks).toBeGreaterThan(0);
  await page.keyboard.press('Escape');
});

test('binding capture rejects conflicts and composition, then Escape cancels safely',async({page})=>{
  await page.goto('/abyss/rift');await page.locator('#rift-controls-open').click();
  await page.locator('[data-remap="attack"]').click();await page.keyboard.press('w');
  await expect(page.locator('#rift-binding-status')).toContainText('Move up');
  await page.evaluate(()=>document.activeElement.dispatchEvent(new KeyboardEvent('keydown',{code:'KeyF',key:'f',isComposing:true,bubbles:true})));
  await expect(page.locator('[data-remap="attack"]')).toContainText('Press a key');
  await page.keyboard.press('Escape');await expect(page.locator('[data-remap="attack"]')).toContainText('J');
  await expect(page.locator('#rift-controls-dialog')).toBeVisible();
  await page.keyboard.press('Escape');await expect(page.locator('#rift-controls-dialog')).toBeHidden();
  await expect(page.locator('#rift-controls-open')).toBeFocused();
});

test('opening controls pauses a fight, confines focus and never fires captured keys',async({page})=>{
  await page.goto('/abyss/rift');await page.locator('#rift-start').click();
  await page.locator('#rift-controls-open').click();
  await expect(page.locator('#rift-controls-dialog')).toBeVisible();
  const read=async()=>(await(await page.request.get('/api/abyss/rift')).json()).run;
  const before=await read();expect(before.paused).toBe(true);
  await page.locator('[data-remap="skill0"]').click();await page.keyboard.press('j');await page.keyboard.press('Escape');
  await page.locator('#rift-controls-close').focus();await page.keyboard.press('Tab');
  expect(await page.evaluate(()=>document.querySelector('#rift-controls-dialog').contains(document.activeElement))).toBe(true);
  await page.locator('#rift-controls-close').click();
  const after=await read();expect(after.paused).toBe(true);expect(after.stats.attacks).toBe(before.stats.attacks);
});

test('alternate layouts and reset are explicit and malformed saved keys recover',async({page})=>{
  await page.addInitScript(()=>localStorage.setItem('riftBindings',JSON.stringify({version:1,bindings:{attack:['KeyF'],left:['KeyF'],jump:['Bogus'],right:['KeyH']}})));
  await page.goto('/abyss/rift');await page.locator('#rift-controls-open').click();
  await expect(page.locator('[data-remap="jump"]')).toContainText('Space');
  await expect(page.locator('[data-remap="right"]')).toContainText('H');
  await page.locator('#rift-key-preset').selectOption('leftHanded');
  await expect(page.locator('[data-remap="right"]')).toContainText('H');
  await page.locator('#rift-apply-keys').click();await expect(page.locator('[data-remap="right"]')).toContainText('ArrowRight');
  await page.locator('#rift-reset-keys').click();await expect(page.locator('[data-remap="attack"]')).toContainText('J');
  await expect(page.locator('[data-remap="right"]')).toContainText('D');
});
