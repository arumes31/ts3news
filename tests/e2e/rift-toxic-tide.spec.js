const {test,expect}=require('@playwright/test');
for(const reduced of [false,true])test(`toxic tide moving pool reduced=${reduced}`,async({page},info)=>{
 const errors=[];page.on('pageerror',e=>errors.push(e.message));await page.goto('/abyss/rift?practice=toxic_tide');await expect(page.locator('#rift-practice-title')).toHaveText('Toxic Tide challenge');await expect(page.locator('#rift-practice-instructions')).toContainText('three moving toxic puddles');await expect(page.locator('#rift-pickup-training')).toBeHidden();await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();await page.keyboard.press('Escape');await expect(page.locator('#rift-paused-badge')).toHaveText('Paused');
 const run=(await(await page.request.get('/api/abyss/rift?practice=toxic_tide')).json()).run;
 await page.evaluate(async({run,reduced})=>{
  await RiftRenderer.ready;RiftRenderer.reduced=reduced;RiftDisplay.hazardLabels=true;RiftDisplay.cameraSmooth=false;
  const ctx=document.querySelector('#rift-canvas').getContext('2d'),rect=ctx.strokeRect;window.poolBoxes=[];
  ctx.strokeRect=function(...args){if(this.strokeStyle==='#dbef9c'&&args[2]===80)poolBoxes.push(args[0]);return rect.apply(this,args);};
  run.clock=1.2;run.player.x=520;run.player.y=400;run.player.jump=0;run.skill_timers={};run.events=[];run.practice.arena.hazards=run.practice.arena.hazards.slice(0,1);window.poolRun=run;document.querySelector('#rift-overlay').hidden=true;RiftRenderer.snapshot(run,true);
 },{run,reduced});
 await expect.poll(()=>page.evaluate(()=>poolBoxes.length>0)).toBe(true);const first=await page.evaluate(()=>poolBoxes.at(-1));expect(await page.evaluate(()=>RiftRenderer.lastPickupRadius)).toBeNull();
 expect(await page.evaluate(()=>RiftHUD.detectPlayerAreaEffects(poolRun)?.kind==='moving_poison')).toBe(false);
 await page.evaluate(()=>{poolRun.clock=3.1;poolBoxes=[];RiftRenderer.snapshot(poolRun,true);RiftMinimap.update(poolRun);});await expect.poll(()=>page.evaluate(()=>poolBoxes.length>0)).toBe(true);
 expect(await page.evaluate(()=>poolBoxes.at(-1))).toBeCloseTo(first+160,5);
 expect(await page.evaluate(()=>{const r=structuredClone(poolRun);r.player.x=580;return RiftHUD.detectPlayerAreaEffects(r)?.kind;})).toBe('moving_poison');
 await expect(page.locator('#rift-minimap [data-kind=hazard]')).toHaveAttribute('width','16');await page.locator('#rift-canvas').screenshot({path:info.outputPath('toxic-pool.png')});
 await page.evaluate(()=>{poolRun.clock=4.05;poolBoxes=[];RiftRenderer.snapshot(poolRun,true);});await expect.poll(()=>page.evaluate(()=>poolBoxes.length>0)).toBe(true);expect(await page.evaluate(()=>poolBoxes.at(-1))).toBeCloseTo(first+80,5);expect(errors).toEqual([]);
});
