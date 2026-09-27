const {test,expect}=require('@playwright/test');
for(const reduced of [false,true])test(`blade circuit keeps a safe hub reduced=${reduced}`,async({page},info)=>{
 const errors=[];page.on('pageerror',e=>errors.push(e.message));await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
 const data=await(await page.request.get('/api/abyss/rift')).json();
 await page.evaluate(async({run,reduced})=>{
  await RiftRenderer.ready;RiftRenderer.reduced=reduced;RiftDisplay.hazardLabels=true;RiftDisplay.cameraSmooth=false;
  const ctx=document.querySelector('#rift-canvas').getContext('2d'),fill=ctx.fillText,rect=ctx.strokeRect;window.bladeLabels=[];window.bladeBoxes=[];
  ctx.fillText=function(text,...args){if(/BLADE|SAFE CENTRE/.test(text))bladeLabels.push(text);return fill.call(this,text,...args);};
  ctx.strokeRect=function(...args){if(this.strokeStyle==='#ffe8b8'&&args[2]===20&&args[3]===20)bladeBoxes.push(args.slice(0,2));return rect.apply(this,args);};
  run.status='fighting';run.paused=true;run.clock=.6;run.events=[];run.player.x=570;run.player.y=385;run.player.jump=0;run.skill_timers={};run.enemies=[];
  run.level.rooms[run.room].hazards=[{x:500,y:350,w:140,h:70,kind:'rotating_blade',period:7,offset:0,duration:2.4,jumpable:true}];
  window.bladeRun=run;document.querySelector('#rift-overlay').hidden=true;RiftRenderer.snapshot(run,true);
 },{run:data.run,reduced});
 await expect.poll(()=>page.evaluate(()=>bladeLabels.includes('CLOCKWISE BLADE IN 0.6s')&&bladeLabels.includes('SAFE CENTRE · JUMP TO CROSS'))).toBe(true);
 await page.evaluate(()=>{bladeRun.clock=1.2;bladeBoxes=[];RiftRenderer.snapshot(bladeRun,true);});await expect.poll(()=>page.evaluate(()=>bladeBoxes.length>0)).toBe(true);
 const first=await page.evaluate(()=>bladeBoxes.at(-1));
 await page.evaluate(()=>{bladeRun.clock=1.8;bladeBoxes=[];RiftRenderer.snapshot(bladeRun,true);RiftMinimap.update(bladeRun);});await expect.poll(()=>page.evaluate(()=>bladeBoxes.length>0)).toBe(true);
 const second=await page.evaluate(()=>bladeBoxes.at(-1));expect(second[0]).toBeCloseTo(first[0]-60,5);expect(second[1]).toBeCloseTo(first[1]+25,5);
 expect(await page.evaluate(()=>RiftHUD.detectPlayerAreaEffects(bladeRun)?.kind==='rotating_blade')).toBe(false);
 expect(await page.evaluate(()=>{const run=structuredClone(bladeRun);run.player.y=410;return RiftHUD.detectPlayerAreaEffects(run)?.kind;})).toBe('rotating_blade');
 await expect(page.locator('#rift-minimap [data-kind=hazard]')).toHaveAttribute('width','4');
 await page.locator('#rift-canvas').screenshot({path:info.outputPath('rotating-blade.png')});
 await page.evaluate(()=>{bladeRun.clock=4;bladeBoxes=[];RiftRenderer.snapshot(bladeRun,true);});await page.waitForTimeout(120);expect(await page.evaluate(()=>bladeBoxes)).toEqual([]);expect(errors).toEqual([]);
});
