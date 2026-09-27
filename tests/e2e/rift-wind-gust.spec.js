const {test,expect}=require('@playwright/test');
for(const reduced of [false,true])test(`wind direction and timed cue reduced=${reduced}`,async({page},info)=>{
 const errors=[];page.on('pageerror',e=>errors.push(e.message));await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
 const data=await(await page.request.get('/api/abyss/rift')).json();
 await page.evaluate(async({run,reduced})=>{
  await RiftRenderer.ready;RiftRenderer.reduced=reduced;RiftDisplay.hazardLabels=true;RiftDisplay.cameraSmooth=false;
  const ctx=document.querySelector('#rift-canvas').getContext('2d'),fill=ctx.fillText;window.windLabels=[];
  ctx.fillText=function(text,...args){if(/^WIND/.test(text))windLabels.push(text);return fill.call(this,text,...args);};
  run.status='fighting';run.paused=true;run.clock=.5;run.events=[];run.player.x=480;run.enemies=[];
  run.level.rooms[run.room].hazards=[];run.level.rooms[run.room].wind_gusts=[{x:420,y:330,w:720,h:150,period:8,offset:0,duration:3,vx:60}];
  window.windRun=run;document.querySelector('#rift-overlay').hidden=true;RiftRenderer.snapshot(run,true);
 },{run:data.run,reduced});
 await expect.poll(()=>page.evaluate(()=>windLabels.includes('WIND IN 0.7s · SHOTS →'))).toBe(true);
 await page.evaluate(()=>{windRun.clock=1.5;windLabels=[];RiftRenderer.snapshot(windRun,true);});
 await expect.poll(()=>page.evaluate(()=>windLabels.includes('WIND 2.7s · SHOTS →'))).toBe(true);
 await page.locator('#rift-canvas').screenshot({path:info.outputPath('wind-active.png')});
 await page.evaluate(()=>{windRun.level.rooms[windRun.room].wind_gusts[0].vx=-60;windLabels=[];RiftRenderer.snapshot(windRun,true);});
 await expect.poll(()=>page.evaluate(()=>windLabels.includes('WIND 2.7s · SHOTS ←'))).toBe(true);
 await page.evaluate(()=>{windRun.clock=5;windLabels=[];RiftRenderer.snapshot(windRun,true);});await page.waitForTimeout(120);expect(await page.evaluate(()=>windLabels)).toEqual([]);await page.evaluate(()=>{windRun.clock=.5;windRun.status='cleared';windLabels=[];RiftRenderer.snapshot(windRun,true);});await page.waitForTimeout(120);expect(await page.evaluate(()=>windLabels)).toEqual([]);expect(errors).toEqual([]);
});
