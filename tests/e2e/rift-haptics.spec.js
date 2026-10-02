const {test,expect}=require('@playwright/test');
async function setup(page){
  await page.addInitScript(()=>{
    window.rumbleCalls=[];window.rumbleResets=0;window.rejectRumble=false;window.rumblePending=[];window.rejectReset=false;
    const actuator={effects:['dual-rumble'],playEffect:(type,params)=>{window.rumbleCalls.push({type,params,at:performance.now()});return window.rejectRumble?Promise.reject(new Error('Unavailable')):new Promise(resolve=>window.rumblePending.push(resolve));},reset:()=>{window.rumbleResets++;return window.rejectReset?Promise.reject(new Error('Hidden')):Promise.resolve('complete');}};
    window.testPad={id:'Haptics test controller',index:0,connected:true,mapping:'standard',axes:[0,0],buttons:Array.from({length:17},()=>({pressed:false,value:0})),vibrationActuator:actuator};Object.defineProperty(navigator,'getGamepads',{value:()=>window.testPad?[window.testPad]:[]});
  });
  await page.goto('/abyss/rift');await page.locator('.rift-settings > summary').click();await page.locator('#rift-gamepad-panel > summary').click();await expect(page.locator('#rift-vibration')).not.toBeChecked();await page.locator('#rift-vibration').check();await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();await page.keyboard.press('Escape');await expect(page.locator('#rift-overlay')).toBeVisible();
  return (await(await page.request.get('/api/abyss/rift')).json()).run;
}

test('confirmed damage produces bounded distinct rumble, rate limits bursts and never replays',async({page})=>{
  const run=await setup(page);const result=await page.evaluate(async run=>{
    run.paused=false;const h=window.RiftHaptics;h.update(run,true,true);run.stats.damage_dealt+=10;h.update(run,false,true);h.update(run,false,true);run.stats.damage_dealt+=10;h.update(run,false,true);
    await new Promise(resolve=>setTimeout(resolve,150));run.stats.damage_taken+=5;h.update(run,false,true);const calls=window.rumbleCalls.slice();window.rumblePending[0]('preempted');await Promise.resolve();run.paused=true;h.update(run,false,false);run.paused=false;h.update(run,true,true);return {calls,resets:window.rumbleResets};
  },run);
  expect(result.calls).toHaveLength(2);for(const call of result.calls){expect(call.type).toBe('dual-rumble');expect(call.params.duration).toBeLessThanOrEqual(120);expect(call.params.startDelay).toBe(0);expect(call.params.strongMagnitude).toBeLessThanOrEqual(1);expect(call.params.weakMagnitude).toBeLessThanOrEqual(1);}expect(result.calls[1].params.strongMagnitude).toBeGreaterThan(result.calls[0].params.strongMagnitude);expect(result.resets).toBeGreaterThan(0);
  await page.reload();await expect(page.locator('#rift-vibration')).toBeChecked();
});

test('hiding the page cancels vibration even when the actuator rejects reset',async({page})=>{
  const errors=[];page.on('pageerror',error=>errors.push(error.message));const run=await setup(page);
  const resets=await page.evaluate(async run=>{run.paused=false;window.RiftHaptics.update(run,true,true);run.stats.damage_taken++;window.RiftHaptics.update(run,false,true);window.rejectReset=true;Object.defineProperty(document,'hidden',{configurable:true,value:true});document.dispatchEvent(new Event('visibilitychange'));await new Promise(resolve=>setTimeout(resolve,20));return window.rumbleResets;},run);
  expect(resets).toBeGreaterThan(0);expect(errors).toEqual([]);
});

test('confirmed responses trigger vibration and disconnect cancels it without replay',async({page})=>{
  await setup(page);await page.route('**/api/abyss/rift',async route=>{const response=await route.fetch(),data=await response.json();if(route.request().postDataJSON()?.kind==='step')data.run.stats.damage_dealt+=10;await route.fulfill({response,json:data});});
  await page.locator('#rift-start').click();await expect.poll(()=>page.evaluate(()=>window.rumbleCalls.length)).toBe(1);
  const resets=await page.evaluate(()=>window.rumbleResets);await page.evaluate(()=>{window.testPad=null;const event=new Event('gamepaddisconnected');event.gamepad={index:0};window.dispatchEvent(event);});await expect(page.locator('#rift-overlay')).toBeVisible();expect(await page.evaluate(()=>window.rumbleResets)).toBeGreaterThan(resets);
});

test('rejected or unsupported vibration cannot break input and disabling cancels active effects',async({page})=>{
  const errors=[];page.on('pageerror',error=>errors.push(error.message));const run=await setup(page);
  await page.evaluate(async run=>{window.rejectRumble=true;run.paused=false;window.RiftHaptics.update(run,true,true);run.stats.damage_taken++;window.RiftHaptics.update(run,false,true);await new Promise(resolve=>setTimeout(resolve,150));window.rejectRumble=false;run.stats.damage_taken++;window.RiftHaptics.update(run,false,true);},run);
  await page.locator('#rift-vibration').uncheck();expect(await page.evaluate(()=>window.rumbleResets)).toBeGreaterThan(0);await page.locator('#rift-vibration').check();await page.evaluate(()=>window.testPad.vibrationActuator=null);await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();await page.keyboard.press('Space');await expect.poll(async()=>(await(await page.request.get('/api/abyss/rift')).json()).run.stats.jumps).toBe(1);await page.keyboard.press('Escape');expect(errors).toEqual([]);
});


test('sustained hit storms stay rate-limited and opt-out suppresses further pulses',async({page})=>{
 const run=await setup(page);
 const result=await page.evaluate(run=>{
  const original=performance.now;let now=original.call(performance)+1000;
  performance.now=()=>now;
  try{
   const h=window.RiftHaptics;run.paused=false;h.update(run,true,true);window.rumbleCalls=[];
   for(let i=0;i<1000;i++){now++;run.stats.damage_taken++;run.stats.damage_dealt++;h.update(run,false,true);}
   const calls=window.rumbleCalls.slice();
   const toggle=document.getElementById('rift-vibration');toggle.checked=false;toggle.dispatchEvent(new Event('change'));
   for(let i=0;i<1000;i++){now+=121;run.stats.damage_taken++;h.update(run,false,true);}
   return {calls,afterDisabled:window.rumbleCalls.length};
  }finally{performance.now=original;}
 },run);
 expect(result.calls).toHaveLength(9);expect(result.afterDisabled).toBe(9);
 for(let i=1;i<result.calls.length;i++)expect(result.calls[i].at-result.calls[i-1].at).toBeGreaterThanOrEqual(120);
});
