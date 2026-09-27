const {test,expect}=require('@playwright/test');
for(const reduced of [false,true])test(`hazard switch indicates charge and shutdown reduced=${reduced}`,async({page},info)=>{
 const errors=[];page.on('pageerror',e=>errors.push(e.message));await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
 const data=await(await page.request.get('/api/abyss/rift')).json();
 await page.evaluate(async({run,reduced})=>{
  await RiftRenderer.ready;RiftRenderer.reduced=reduced;RiftDisplay.hazardLabels=true;RiftDisplay.cameraSmooth=false;
  const ctx=document.querySelector('#rift-canvas').getContext('2d'),fill=ctx.fillText,rect=ctx.fillRect;window.switchLabels=[];window.switchBars=[];
  ctx.fillText=function(text,...args){if(/SHUT DOWN|HAZARDS OFF|SWITCH INACTIVE/.test(text))switchLabels.push(text);return fill.call(this,text,...args);};
  ctx.fillRect=function(...args){if(this.fillStyle==='#a4efd0'&&args[3]===4)switchBars.push(args[2]);return rect.apply(this,args);};
  run.status='fighting';run.paused=true;run.clock=1.5;run.events=[];run.player.x=450;run.player.y=400;run.enemies=[];
  run.level.rooms[run.room].hazard_switch={x:370,y:475,charge:.3,used:false};run.level.rooms[run.room].hazards=[{x:600,y:350,w:90,h:40,kind:'fire',period:7,offset:0,duration:1,jumpable:true}];
  window.switchRun=run;document.querySelector('#rift-overlay').hidden=true;RiftRenderer.snapshot(run,true);RiftMinimap.update(run);
 },{run:data.run,reduced});
 await expect.poll(()=>page.evaluate(()=>switchLabels.includes('HOLD GUARD · SHUT DOWN')&&switchBars.includes(18))).toBe(true);
 await expect(page.locator('#rift-minimap [data-kind=switch]')).toHaveCount(1);
 await expect(page.locator('#rift-minimap [data-kind=switch]')).toHaveCSS('fill','rgb(228, 189, 118)');
 await page.locator('#rift-canvas').screenshot({path:info.outputPath('switch-charge.png')});
 await page.evaluate(()=>{switchRun.level.rooms[switchRun.room].hazard_switch={x:370,y:475,charge:.6,used:true};switchRun.level.rooms[switchRun.room].hazards[0].disabled=true;switchLabels=[];switchBars=[];RiftRenderer.snapshot(switchRun,true);RiftMinimap.update(switchRun);});
 await expect.poll(()=>page.evaluate(()=>switchLabels.includes('HAZARDS OFF'))).toBe(true);expect(await page.evaluate(()=>switchBars)).toEqual([]);
 await expect(page.locator('#rift-minimap [data-kind=hazard]')).toHaveAttribute('data-phase','off');await page.evaluate(()=>{switchRun.status='cleared';switchRun.level.rooms[switchRun.room].hazard_switch.used=false;switchLabels=[];RiftRenderer.snapshot(switchRun,true);});await expect.poll(()=>page.evaluate(()=>switchLabels.includes('SWITCH INACTIVE'))).toBe(true);expect(errors).toEqual([]);
});
