const {test,expect}=require('@playwright/test');
for(const reduced of [false,true])test(`flame sweep matches moving damage strip reduced=${reduced}`,async({page},info)=>{
 const errors=[];page.on('pageerror',e=>errors.push(e.message));await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
 const data=await(await page.request.get('/api/abyss/rift')).json();
 await page.evaluate(async({run,reduced})=>{
  await RiftRenderer.ready;RiftRenderer.reduced=reduced;RiftDisplay.hazardLabels=true;RiftDisplay.cameraSmooth=false;
  const ctx=document.querySelector('#rift-canvas').getContext('2d'),fill=ctx.fillText,rect=ctx.strokeRect;window.sweepLabels=[];window.sweepStrips=[];
  ctx.fillText=function(text,...args){if(/SWEEP|PASS BEHIND/.test(text))sweepLabels.push(text);return fill.call(this,text,...args);};
  ctx.strokeRect=function(...args){if(this.strokeStyle==='#ffe5ae'&&args[2]===36)sweepStrips.push(args[0]);return rect.apply(this,args);};
  run.status='fighting';run.paused=true;run.clock=.6;run.events=[];run.player.x=620;run.player.y=370;run.player.jump=0;run.skill_timers={};run.enemies=[];
  run.level.rooms[run.room].hazards=[{x:500,y:350,w:240,h:40,kind:'sweeping_flame',period:7,offset:0,duration:1.8,jumpable:true}];
  window.sweepRun=run;document.querySelector('#rift-overlay').hidden=true;RiftRenderer.snapshot(run,true);
 },{run:data.run,reduced});
 await expect.poll(()=>page.evaluate(()=>sweepLabels.includes('SWEEP RIGHT IN 0.6s')&&sweepLabels.includes('JUMP OR PASS BEHIND'))).toBe(true);
 await page.evaluate(()=>{sweepRun.clock=1.21;sweepStrips=[];RiftRenderer.snapshot(sweepRun,true);RiftMinimap.update(sweepRun);});
 await expect.poll(()=>page.evaluate(()=>sweepStrips.length>0)).toBe(true);
 expect(await page.evaluate(()=>RiftHUD.detectPlayerAreaEffects(sweepRun)?.kind==='sweeping_flame')).toBe(false);
 const first=await page.evaluate(()=>sweepStrips.at(-1));
 await page.evaluate(()=>{sweepRun.clock=2.1;sweepStrips=[];RiftRenderer.snapshot(sweepRun,true);RiftMinimap.update(sweepRun);});
 await expect.poll(()=>page.evaluate(()=>sweepStrips.length>0)).toBe(true);
 expect(await page.evaluate(()=>sweepStrips.at(-1))).toBeCloseTo(first+100.8666667,4);
 expect(await page.evaluate(()=>RiftHUD.detectPlayerAreaEffects(sweepRun)?.kind)).toBe('sweeping_flame');
 await expect(page.locator('#rift-minimap [data-kind=hazard]')).toHaveAttribute('width','7.2');
 await page.locator('#rift-canvas').screenshot({path:info.outputPath('flame-sweep.png')});
 await page.evaluate(()=>{sweepRun.clock=3.1;sweepStrips=[];RiftRenderer.snapshot(sweepRun,true);});await page.waitForTimeout(120);expect(await page.evaluate(()=>sweepStrips)).toEqual([]);expect(errors).toEqual([]);
});
