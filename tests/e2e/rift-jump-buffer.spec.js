const {test,expect}=require('@playwright/test');
const read=async page=>(await(await page.request.get('/api/abyss/rift')).json()).run;

for(const buffer of ['0','300'])test('early jump request with '+buffer+'ms buffer uses confirmed server recovery',async({page})=>{
  await page.goto('/abyss/rift');await page.locator('.rift-settings > summary').click();await expect(page.locator('#rift-jump-buffer')).toHaveValue('0');await page.locator('#rift-jump-buffer').selectOption(buffer);
  await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();
  let notify,phase=0;const pressed=new Promise(resolve=>notify=resolve);
  await page.route('**/api/abyss/rift',async route=>{
    if(route.request().postDataJSON()?.kind!=='step'){await route.continue();return;}
    const response=await route.fetch(),data=await response.json(),remaining=data.run.skill_timers.jump;
    if(notify&&data.run.stats.jumps===1&&remaining>.15&&remaining<.30){
      phase=remaining;const ready=notify;notify=null;
      // Observe a wider band than one network tick, then place the press
      // about 190 ms before recovery instead of waiting for an exact snapshot.
      await new Promise(resolve=>setTimeout(resolve,Math.max(0,(remaining-.19)*1000)));
      await page.keyboard.press('Space');ready();
    }
    await route.fulfill({response,json:data});
  });
  await page.keyboard.press('Space');await pressed;expect(phase).toBeGreaterThan(.15);
  if(buffer==='300')await expect.poll(async()=>(await read(page)).stats.jumps).toBe(2);
  else{await page.waitForTimeout(550);expect((await read(page)).stats.jumps).toBe(1);}
  await page.waitForTimeout(450);expect((await read(page)).stats.jumps).toBe(buffer==='300'?2:1);await page.keyboard.press('Escape');
});

test('confirmed jumps, pause and pointer cancellation clear buffered requests',async({page})=>{
  await page.goto('/abyss/rift');await page.locator('.rift-settings > summary').click();await page.locator('#rift-jump-buffer').selectOption('300');await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();await page.keyboard.press('Escape');await expect(page.locator('#rift-overlay')).toBeVisible();const run=await read(page);
  const lifecycle=await page.evaluate(run=>{
    const buffer=window.RiftJump;run.paused=false;buffer.sync(run,true);buffer.press();const before=buffer.input(false);run.stats.jumps++;buffer.sync(run,false);const confirmed=buffer.input(false);buffer.press();run.paused=true;buffer.sync(run,false);return {before,confirmed,paused:buffer.input(false)};
  },run);expect(lifecycle).toEqual({before:true,confirmed:false,paused:false});
  await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();let release,notify;const blocked=new Promise(resolve=>notify=resolve);
  await page.route('**/api/abyss/rift',async route=>{if(route.request().postDataJSON()?.kind==='step'&&notify){const ready=notify;notify=null;await new Promise(resolve=>{release=resolve;ready();});}await route.continue();});
  await blocked;const button=page.locator('[data-bind="jump"]');await button.hover();await page.mouse.down();expect(await page.evaluate(()=>window.RiftJump.input(false))).toBe(true);await button.dispatchEvent('pointercancel',{pointerId:1});await page.mouse.up();expect(await page.evaluate(()=>window.RiftJump.input(false))).toBe(false);release();await page.keyboard.press('Escape');await expect(page.locator('#rift-overlay')).toBeVisible();
  await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();await page.waitForTimeout(400);expect((await read(page)).stats.jumps).toBe(0);await page.keyboard.press('Escape');
});

test('jump buffer preference persists, validates saved values and resets pending input on pause',async({page})=>{
  await page.goto('/abyss/rift');await page.locator('.rift-settings > summary').click();await page.locator('#rift-jump-buffer').selectOption('200');await page.reload();await expect(page.locator('#rift-jump-buffer')).toHaveValue('200');
  const result=await page.evaluate(()=>{const buffer=window.RiftJump;buffer.press();const pending=buffer.input(false);buffer.reset();return {pending,cleared:buffer.input(false),held:buffer.input(true)};});expect(result).toEqual({pending:true,cleared:false,held:true});
  await page.evaluate(()=>localStorage.setItem('riftJumpBuffer','999'));await page.reload();await expect(page.locator('#rift-jump-buffer')).toHaveValue('0');
});
