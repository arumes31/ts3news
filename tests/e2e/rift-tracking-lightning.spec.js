const {test,expect}=require('@playwright/test');
for(const reduced of [false,true])test('lightning tracking lock and strike reduced='+reduced,async({page},info)=>{
 const errors=[];page.on('pageerror',e=>errors.push(e.message));await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
 const data=await(await page.request.get('/api/abyss/rift')).json();
 await page.evaluate(async({run,reduced})=>{
  await RiftRenderer.ready;RiftRenderer.reduced=reduced;RiftDisplay.cameraSmooth=false;RiftDisplay.hazardLabels=true;
  const ctx=document.querySelector('#rift-canvas').getContext('2d'),fill=ctx.fillText;window.lightningLabels=[];
  ctx.fillText=function(text,...args){if(/TRACKING|LOCKED|LIGHTNING/.test(text))lightningLabels.push(text);return fill.call(this,text,...args);};
  run.status='fighting';run.paused=true;run.clock=.2;run.events=[];run.player.x=480;run.enemies=[];
  run.level.rooms[run.room].hazards=[{x:540,y:400,w:84,h:48,kind:'tracking_lightning',period:7,offset:0,duration:.18,jumpable:false}];
  window.lightningRun=run;document.querySelector('#rift-overlay').hidden=true;RiftRenderer.snapshot(run,true);RiftMinimap.update(run);
 },{run:data.run,reduced});
 await expect.poll(()=>page.evaluate(()=>lightningLabels.includes('TRACKING · KEEP MOVING'))).toBe(true);
 await page.evaluate(()=>{lightningRun.clock=.7;lightningLabels=[];RiftRenderer.snapshot(lightningRun,true);});
 await expect.poll(()=>page.evaluate(()=>lightningLabels.includes('LOCKED · MOVE OUT'))).toBe(true);
 await page.locator('#rift-canvas').screenshot({path:info.outputPath('lightning-locked.png')});
 await page.evaluate(()=>{lightningRun.clock=1.22;lightningLabels=[];RiftRenderer.snapshot(lightningRun,true);RiftMinimap.update(lightningRun);});
 await expect.poll(()=>page.evaluate(()=>lightningLabels.includes('LIGHTNING · MOVE'))).toBe(true);
 await expect(page.locator('#rift-minimap [data-phase=active]')).toHaveAttribute('x','108');
 await page.locator('#rift-canvas').screenshot({path:info.outputPath('lightning-strike.png')});
 await page.evaluate(()=>{lightningRun.clock=2;lightningLabels=[];RiftRenderer.snapshot(lightningRun,true);});await page.waitForTimeout(120);expect(await page.evaluate(()=>lightningLabels)).toEqual([]);expect(errors).toEqual([]);
});
