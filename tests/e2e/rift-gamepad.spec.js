const {test,expect}=require('@playwright/test');

async function controller(page){
  await page.addInitScript(()=>{
    window.testPad={id:'Brawl standard test controller',index:0,mapping:'standard',connected:true,axes:[0,0,0,0],buttons:Array.from({length:17},()=>({pressed:false,value:0}))};
    Object.defineProperty(navigator,'getGamepads',{configurable:true,value:()=>window.testPad?[window.testPad]:[]});
  });
  await page.goto('/abyss/rift');await expect(page.locator('#rift-gamepad-status')).toContainText('Connected:');
  await expect(page.locator('#rift-gamepad-status')).not.toContainText('Release controls');
}
async function input(page,buttons=[],axes=[0,0]){await page.evaluate(({buttons,axes})=>{window.testPad.axes=axes;window.testPad.buttons=Array.from({length:17},(_,i)=>({pressed:buttons.includes(i),value:buttons.includes(i)?1:0}));},{buttons,axes});}
async function tap(page,button){await input(page,[button]);await page.waitForTimeout(65);await input(page);await page.waitForTimeout(65);}
const read=async page=>(await(await page.request.get('/api/abyss/rift')).json()).run;

test('controller panel shows live readings and saves independent valid stick settings',async({page})=>{
  await controller(page);await page.locator('.rift-settings > summary').click();await page.locator('#rift-gamepad-panel > summary').click();
  for(const [key,value] of [['horizontal','0.5'],['vertical','1.5'],['deadzone','0.2']])await page.locator('#rift-gamepad-'+key).evaluate((node,value)=>{node.value=value;node.dispatchEvent(new Event('input',{bubbles:true}));},value);
  await input(page,[],[.6,.6]);await expect(page.locator('#rift-gamepad-test')).toContainText('X 0.25 · Y 0.75');
  await input(page,[],[.1,-.1]);await expect(page.locator('#rift-gamepad-test')).toContainText('X 0.00 · Y 0.00');
  await page.reload();await page.locator('.rift-settings > summary').click();await page.locator('#rift-gamepad-panel > summary').click();
  await expect(page.locator('#rift-gamepad-horizontal')).toHaveValue('0.5');await expect(page.locator('#rift-gamepad-vertical')).toHaveValue('1.5');await expect(page.locator('#rift-gamepad-deadzone')).toHaveValue('0.2');
});

test('standard controller sends analog movement and all combat bindings to the server',async({page})=>{
  const steps=[];page.on('request',request=>{if(request.method()==='POST'&&request.url().endsWith('/api/abyss/rift')){const body=request.postDataJSON();if(body.kind==='step')steps.push(body.input);}});
  await controller(page);await page.locator('#rift-start').click();const run=await read(page),x=run.player.x;
  await input(page,[],[.59,0]);await expect.poll(()=>steps.some(i=>Math.abs(i.x-.5)<.001)).toBe(true);await expect.poll(async()=>(await read(page)).player.x).toBeGreaterThan(x);await input(page);
  async function binding(button,check){steps.length=0;await input(page,[button]);await expect.poll(()=>steps.some(check)).toBe(true);await input(page);await page.waitForTimeout(70);}
  await binding(2,i=>i.attack);await expect.poll(async()=>(await read(page)).stats.attacks).toBeGreaterThan(0);
  await binding(1,i=>i.guard);await binding(0,i=>i.jump);
  await binding(4,i=>i.skill===run.build.signatures[0].id);await binding(5,i=>i.skill===run.build.signatures[1].id);
  await binding(8,i=>i.skill===run.build.ultimate.id);await binding(3,i=>i.skill===run.build.skills[0].id);
  await tap(page,7);await binding(3,i=>i.skill===run.build.skills[1].id);await tap(page,6);await binding(3,i=>i.skill===run.build.skills[0].id);
  await binding(14,i=>i.x===-1);await binding(13,i=>i.y===1);await tap(page,9);await expect.poll(async()=>(await read(page)).paused).toBe(true);
});

test('disconnect pauses and reconnecting held controls cannot resume stale attacks',async({page})=>{
  await controller(page);await page.locator('#rift-start').click();await input(page,[2],[1,0]);await expect.poll(async()=>(await read(page)).stats.attacks).toBeGreaterThan(0);
  await page.evaluate(()=>{window.disconnectedPad=window.testPad;window.testPad=null;const event=new Event('gamepaddisconnected');event.gamepad={index:0};window.dispatchEvent(event);});
  await expect.poll(async()=>(await read(page)).paused).toBe(true);const attacks=(await read(page)).stats.attacks;
  await page.evaluate(()=>window.testPad=window.disconnectedPad);await expect(page.locator('#rift-gamepad-status')).toContainText('Release controls');
  await page.locator('#rift-start').click();await page.waitForTimeout(350);expect((await read(page)).stats.attacks).toBe(attacks);
  await input(page);await expect(page.locator('#rift-gamepad-status')).not.toContainText('Release controls');await input(page,[],[1,0]);await expect(page.locator('#rift-movement-keys')).toHaveText('Stick / D-pad');await input(page,[2]);await expect.poll(async()=>(await read(page)).stats.attacks).toBeGreaterThan(attacks);await input(page);await page.keyboard.press('Escape');
});

test('controller menus navigate and activate while paused and remapping remains isolated',async({page})=>{
  await controller(page);await page.locator('.rift-settings > summary').click();const summary=page.locator('#rift-gamepad-panel > summary');await summary.focus();await tap(page,0);await expect(page.locator('#rift-gamepad-panel')).toHaveAttribute('open','');
  await tap(page,13);await expect(page.locator('#rift-gamepad-deadzone')).toBeFocused();await tap(page,15);await expect(page.locator('#rift-gamepad-deadzone')).toHaveValue('0.2');
  await tap(page,9);await expect.poll(async()=>!!(await read(page))&&!((await read(page)).paused)).toBe(true);await tap(page,9);await expect.poll(async()=>(await read(page)).paused).toBe(true);
  await page.locator('#rift-controls-open').click();await page.locator('[data-remap="attack"]').click();await tap(page,2);await tap(page,9);await expect(page.locator('#rift-controls-dialog')).toBeVisible();await expect.poll(async()=>(await read(page)).paused).toBe(true);await expect(page.locator('[data-remap="attack"]')).toHaveText('Press a key…');
});

test('active-device prompts switch back to keyboard and unsupported controller access stays usable',async({page})=>{
  await controller(page);await page.locator('#rift-start').click();await input(page,[2]);await expect(page.locator('[data-bind="attack"] kbd')).toHaveText('West');await input(page);await page.keyboard.press('KeyJ');await expect(page.locator('[data-bind="attack"] kbd')).toHaveText('J');await page.keyboard.press('Escape');
  await page.evaluate(()=>Object.defineProperty(navigator,'getGamepads',{value:()=>{throw new DOMException('Blocked','SecurityError');}}));await expect(page.locator('#rift-gamepad-status')).toContainText('unavailable');await page.locator('#rift-start').click();await expect.poll(async()=>(await read(page)).paused).toBe(false);await page.keyboard.press('Escape');
});

test('sparse controller slots, invalid preferences and controller-only entry stay usable',async({page})=>{
  await page.addInitScript(()=>localStorage.setItem('riftGamepad',JSON.stringify({version:1,deadzone:'0.2',horizontal:999,vertical:.75})));
  await controller(page);await expect(page.locator('#rift-gamepad-deadzone')).toHaveValue('0.18');await expect(page.locator('#rift-gamepad-horizontal')).toHaveValue('1');await expect(page.locator('#rift-gamepad-vertical')).toHaveValue('0.75');
  expect(await read(page)).toBeNull();
  await page.evaluate(()=>{window.testPad.index=2;Object.defineProperty(navigator,'getGamepads',{value:()=>[null,null,window.testPad]});});
  await expect(page.locator('#rift-gamepad-status')).toContainText('Connected:');await expect(page.locator('#rift-gamepad-status')).not.toContainText('Release controls');
  await tap(page,9);await expect.poll(async()=>!!(await read(page))).toBe(true);await input(page);await page.keyboard.press('Escape');
});
