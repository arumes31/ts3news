const {test,expect}=require('@playwright/test');
for(const reduced of [false,true])test(`water current counterplay cue reduced=${reduced}`,async({page},info)=>{
 const errors=[];page.on('pageerror',e=>errors.push(e.message));await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
 const data=await(await page.request.get('/api/abyss/rift')).json();
 await page.evaluate(async({run,reduced})=>{
  await RiftRenderer.ready;RiftRenderer.reduced=reduced;RiftDisplay.hazardLabels=true;RiftDisplay.cameraSmooth=false;
  const ctx=document.querySelector('#rift-canvas').getContext('2d'),fill=ctx.fillText;window.currentLabels=[];
  ctx.fillText=function(text,...args){if(/^CURRENT/.test(text))currentLabels.push(text);return fill.call(this,text,...args);};
  run.status='fighting';run.paused=true;run.clock=1;run.events=[];run.player.x=480;run.enemies=[];
  run.level.rooms[run.room].hazards=[];run.level.rooms[run.room].water_currents=[{x:460,y:400,w:500,h:65,vx:35,vy:0}];
  window.currentRun=run;document.querySelector('#rift-overlay').hidden=true;RiftRenderer.snapshot(run,true);
 },{run:data.run,reduced});
 await expect.poll(()=>page.evaluate(()=>currentLabels.includes('CURRENT · GUARD TO BRACE / JUMP'))).toBe(true);
 await page.locator('#rift-canvas').screenshot({path:info.outputPath('water-current.png')});
 await page.evaluate(()=>{currentRun.status='cleared';currentLabels=[];RiftRenderer.snapshot(currentRun,true);});await page.waitForTimeout(120);expect(await page.evaluate(()=>currentLabels)).toEqual([]);expect(errors).toEqual([]);
});
