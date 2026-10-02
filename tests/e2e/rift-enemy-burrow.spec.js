const {test,expect}=require('@playwright/test');
for(const reduced of [false,true])test(`burrow approach and emergence remain readable reduced=${reduced}`,async({page},info)=>{
 const errors=[];page.on('pageerror',error=>errors.push(error.message));
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
 const data=await(await page.request.get('/api/abyss/rift')).json();const enemy=data.bestiary.find(e=>e.burrowing);expect(enemy).toBeTruthy();
 await page.evaluate(async({run,enemy,reduced})=>{
  await RiftRenderer.ready;RiftRenderer.reduced=reduced;RiftDisplay.enemyIntent=true;
  const ctx=document.querySelector('#rift-canvas').getContext('2d'),fill=ctx.fillText,ellipse=ctx.ellipse;
  window.burrowLabels=[];window.burrowRings=[];window.mounds=[];window.burrowCancelled=false;
  ctx.fillText=function(text,...args){if(text==='BURROW INTERRUPTED')burrowCancelled=true;if(/BURROWING|EMERGING|Burrowing approach|INTENT · Emerging/.test(text))burrowLabels.push(text);return fill.call(this,text,...args);};
  ctx.ellipse=function(...args){if(args[2]===75&&args[3]===40)burrowRings.push(args);if(args[2]===22&&args[3]===9)mounds.push(args);return ellipse.apply(this,args);};
  run.status='fighting';run.paused=true;run.events=[];run.player.x=440;
  run.enemies=[{...enemy,id:'burrower',x:750,y:410,hp:100,max_hp:100,burrowed:true,windup:1.6,target_x:560,target_y:410,attack_name:'Burrow',pose:'windup'}];
  window.burrowRun=run;document.querySelector('#rift-overlay').hidden=true;RiftRenderer.snapshot(run,true);
 },{run:data.run,enemy,reduced});
 await expect.poll(()=>page.evaluate(()=>burrowLabels.includes('BURROWING · HIT TO INTERRUPT · 1.6s')&&burrowLabels.includes('INTENT · Burrowing approach')&&burrowRings.length>0&&mounds.length>0)).toBe(true);
 await page.locator('#rift-canvas').screenshot({path:info.outputPath('burrow-approach.png')});
 await page.evaluate(()=>{Object.assign(burrowRun.enemies[0],{x:560,windup:.8});burrowLabels=[];RiftRenderer.snapshot(burrowRun,true);});
 await expect.poll(()=>page.evaluate(()=>burrowLabels.includes('EMERGING · MOVE OR JUMP · 0.8s')&&burrowLabels.includes('INTENT · Emerging'))).toBe(true);
 await page.locator('#rift-canvas').screenshot({path:info.outputPath('burrow-emergence.png')});
 await page.evaluate(()=>{Object.assign(burrowRun.enemies[0],{burrowed:false,windup:0,pose:'hit',attack_name:''});burrowLabels=[];burrowRings=[];mounds=[];burrowRun.counter++;burrowRun.events=[{id:burrowRun.counter,kind:'burrow_cancel',x:560,y:385,value:0}];RiftRenderer.snapshot(burrowRun,true);});
 await expect.poll(()=>page.evaluate(()=>burrowCancelled)).toBe(true);
 await page.waitForTimeout(120);expect(await page.evaluate(()=>[burrowLabels.length,burrowRings.length,mounds.length])).toEqual([0,0,0]);
 await page.evaluate(()=>{Object.assign(burrowRun.enemies[0],{burrowed:true,windup:.8,attack_name:'Burrow',pose:'windup'});RiftDisplay.cleanScreenshot=true;burrowLabels=[];burrowRings=[];RiftRenderer.snapshot(burrowRun,true);});
 await page.waitForTimeout(120);expect(await page.evaluate(()=>[burrowLabels.length,burrowRings.length])).toEqual([0,0]);
 expect(errors).toEqual([]);
});
