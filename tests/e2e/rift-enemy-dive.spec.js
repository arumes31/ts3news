const {test,expect}=require('@playwright/test');
for(const reduced of [false,true])test(`dive warning path and grounded recovery reduced=${reduced}`,async({page},info)=>{
 const errors=[];page.on('pageerror',e=>errors.push(e.message));
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
 const data=await(await page.request.get('/api/abyss/rift')).json(),enemy=data.bestiary.find(e=>e.flying);expect(enemy).toBeTruthy();
 await page.evaluate(async({run,enemy,reduced})=>{
  await RiftRenderer.ready;RiftRenderer.reduced=reduced;RiftDisplay.enemyIntent=true;
  const ctx=document.querySelector('#rift-canvas').getContext('2d'),fill=ctx.fillText,rect=ctx.strokeRect;
  window.diveLabels=[];window.diveTargets=[];window.diveInterrupted=false;
  ctx.fillText=function(text,...args){if(text==='DIVE INTERRUPTED')diveInterrupted=true;if(/DIVE|SWOOP|Preparing dive|INTENT · Diving|INTENT · Recovering/.test(text))diveLabels.push(text);return fill.call(this,text,...args);};
  ctx.strokeRect=function(...args){if(args[2]===60&&args[3]===40)diveTargets.push(args);return rect.apply(this,args);};
  run.status='fighting';run.paused=true;run.events=[];run.player.x=440;
  run.enemies=[{...enemy,id:'bat',x:750,y:410,hp:100,max_hp:100,windup:1,target_x:560,target_y:410,attack_name:'Dive',pose:'windup'}];
  window.diveRun=run;document.querySelector('#rift-overlay').hidden=true;RiftRenderer.snapshot(run,true);
 },{run:data.run,enemy,reduced});
 await expect.poll(()=>page.evaluate(()=>diveLabels.includes('DIVE · SIDESTEP OR JUMP · 1.0s')&&diveLabels.includes('INTENT · Preparing dive')&&diveTargets.length>0)).toBe(true);
 await page.locator('#rift-canvas').screenshot({path:info.outputPath('dive-warning.png')});
 await page.evaluate(()=>{Object.assign(diveRun.enemies[0],{x:650,windup:0,diving:true,pose:'attack',pose_time:.4});diveLabels=[];RiftRenderer.snapshot(diveRun,true);});
 await expect.poll(()=>page.evaluate(()=>diveLabels.includes('SWOOP · FIXED PATH')&&diveLabels.includes('INTENT · Diving'))).toBe(true);
 await page.evaluate(()=>{Object.assign(diveRun.enemies[0],{x:560,diving:false,attack_name:'',dive_recovery:.8,pose:'recovery',pose_time:0});diveLabels=[];diveTargets=[];RiftRenderer.snapshot(diveRun,true);});
 await expect.poll(()=>page.evaluate(()=>diveLabels.includes('DIVE RECOVERY · 0.8s')&&diveLabels.includes('INTENT · Recovering'))).toBe(true);
 expect(await page.evaluate(()=>diveTargets.length)).toBe(0);
 await page.locator('#rift-canvas').screenshot({path:info.outputPath('dive-recovery.png')});
 await page.evaluate(()=>{diveRun.counter++;diveRun.events=[{id:diveRun.counter,kind:'dive_cancel',x:560,y:385,value:0}];RiftRenderer.snapshot(diveRun,true);});
 await expect.poll(()=>page.evaluate(()=>diveInterrupted)).toBe(true);
 await page.evaluate(()=>{RiftDisplay.cleanScreenshot=true;diveLabels=[];diveTargets=[];RiftRenderer.snapshot(diveRun,true);});
 await page.waitForTimeout(120);expect(await page.evaluate(()=>[diveLabels.length,diveTargets.length])).toEqual([0,0]);expect(errors).toEqual([]);
});
