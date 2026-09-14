const {test,expect}=require('@playwright/test');

async function enable(page){await page.goto('/abyss/rift');await page.locator('.rift-settings > summary').click();await expect(page.locator('#rift-input-readout-enabled')).not.toBeChecked();await page.locator('#rift-input-readout-enabled').check();}

test('optional input readout recognizes keyboard and pointer input without claiming confirmed casts',async({page})=>{
  await enable(page);await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();await page.keyboard.press('KeyD');await expect(page.locator('#rift-input-readout')).toContainText('Move right');
  await page.locator('[data-bind="attack"]').click();await expect(page.locator('#rift-input-readout')).toContainText('Attack');await expect(page.locator('#rift-input-readout')).toHaveAttribute('aria-live','off');
  await page.locator('#rift-canvas').focus();await page.keyboard.down('KeyL');await page.keyboard.press('KeyQ');await expect(page.locator('#rift-input-readout')).toContainText('Release guard');await page.keyboard.up('KeyL');
  await page.keyboard.press('Escape');await expect(page.locator('#rift-input-readout')).toHaveText('Input paused');await page.reload();await expect(page.locator('#rift-input-readout-enabled')).toBeChecked();
});

test('controller movement and buttons are recognized and the readout can stay disabled',async({page})=>{
  await page.addInitScript(()=>{window.testPad={id:'Readout controller',index:0,connected:true,mapping:'standard',axes:[0,0],buttons:Array.from({length:17},()=>({pressed:false,value:0}))};Object.defineProperty(navigator,'getGamepads',{value:()=>[window.testPad]});});
  await enable(page);await expect(page.locator('#rift-gamepad-status')).not.toContainText('Release controls');await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();
  await page.evaluate(()=>window.testPad.axes=[0,-1]);await expect(page.locator('#rift-input-readout')).toContainText('Move up');
  await page.evaluate(()=>{window.testPad.axes=[0,0];window.testPad.buttons[0]={pressed:true,value:1};});await expect(page.locator('#rift-input-readout')).toContainText('Jump');
  await page.evaluate(()=>window.testPad.buttons[0]={pressed:false,value:0});await page.keyboard.press('Escape');await page.locator('#rift-input-readout-enabled').uncheck();await expect(page.locator('#rift-input-readout')).toBeHidden();await page.reload();await expect(page.locator('#rift-input-readout-enabled')).not.toBeChecked();
});

test('blocked queued casts explain mana, cooldown and guard and expire after input stops',async({page})=>{
  await enable(page);await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();await page.keyboard.press('Escape');const run=(await(await page.request.get('/api/abyss/rift')).json()).run;
  async function reason(kind){return page.evaluate(({run,kind})=>{const intent=window.RiftIntents,skill=run.build.signatures[0];run.paused=false;run.player.cooldown=0;run.player.mana=kind==='mana'?0:100;run.skill_timers[skill.id]=kind==='cooldown'?2:0;intent.sync(run,true);intent.press('signature0');intent.take(run,()=>false,kind==='guard');return document.getElementById('rift-input-readout').textContent;},{run,kind});}
  expect(await reason('mana')).toContain('Not enough mana');expect(await reason('cooldown')).toContain('Cooldown');expect(await reason('guard')).toContain('Release guard');
  await expect(page.locator('#rift-input-readout')).toBeHidden({timeout:4000});
});
