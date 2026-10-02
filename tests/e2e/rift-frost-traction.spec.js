const {test,expect}=require('@playwright/test');
for(const reduced of [false,true])test(`frost warns traction and braking reduced=${reduced}`,async({page},info)=>{
 const errors=[];page.on('pageerror',e=>errors.push(e.message));
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
 const data=await(await page.request.get('/api/abyss/rift')).json();
 await page.evaluate(async({run,reduced})=>{
  await RiftRenderer.ready;RiftRenderer.reduced=reduced;RiftDisplay.hazardLabels=true;RiftDisplay.cameraSmooth=false;
  const ctx=document.querySelector('#rift-canvas').getContext('2d'),fill=ctx.fillText;window.frostLabels=[];
  ctx.fillText=function(text,...args){if(/SLIPPERY|JUMP/.test(text))frostLabels.push(text);return fill.call(this,text,...args);};
  run.status='fighting';run.paused=true;run.clock=1.5;run.events=[];run.player.x=480;run.enemies=[];
  run.level.rooms[run.room].hazards=[{x:520,y:400,w:160,h:50,kind:'ice',period:7,offset:0,duration:1,jumpable:true,slippery:true}];
  window.frostRun=run;document.querySelector('#rift-overlay').hidden=true;RiftRenderer.snapshot(run,true);
 },{run:data.run,reduced});
 await expect.poll(()=>page.evaluate(()=>frostLabels.includes('SLIPPERY · GUARD BRAKES')&&frostLabels.includes('JUMP · 0.7s'))).toBe(true);
 await page.locator('#rift-canvas').screenshot({path:info.outputPath('frost-warning.png')});
 await page.evaluate(()=>{frostRun.clock=4;frostLabels=[];RiftRenderer.snapshot(frostRun,true);});await page.waitForTimeout(120);
 expect(await page.evaluate(()=>frostLabels)).toEqual([]);
 await page.evaluate(()=>{frostRun.clock=1.5;frostRun.level.rooms[frostRun.room].hazards[0].disabled=true;frostLabels=[];RiftRenderer.snapshot(frostRun,true);});await page.waitForTimeout(120);
 expect(await page.evaluate(()=>frostLabels)).toEqual([]);expect(errors).toEqual([]);
});
