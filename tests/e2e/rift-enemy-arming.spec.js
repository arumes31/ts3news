const {test,expect}=require('@playwright/test');
test('explosive arming shows exact blast area and interruption guidance',async({page},testInfo)=>{
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
 const data=await(await page.request.get('/api/abyss/rift')).json(),enemy=data.bestiary.find(e=>e.explosive);expect(enemy).toBeTruthy();
 await page.evaluate(async({run,enemy})=>{
  await RiftRenderer.ready;RiftRenderer.reduced=true;RiftDisplay.enemyIntent=true;
  const ctx=document.querySelector('#rift-canvas').getContext('2d'),fill=ctx.fillText,ellipse=ctx.ellipse;
  window.armingLabels=[];window.armingRings=[];window.cancelledBlast=false;
  ctx.fillText=function(text,...args){if(text==='BLAST CANCELLED')cancelledBlast=true;if(/ARMING|Arming blast/.test(text))armingLabels.push(text);return fill.call(this,text,...args);};
  ctx.ellipse=function(...args){if(args[2]===115&&args[3]===55)armingRings.push(args);return ellipse.apply(this,args);};
  run.status='fighting';run.paused=true;run.events=[];run.player.x=500;
  run.enemies=[{...enemy,id:'arming',x:630,y:410,hp:100,max_hp:100,arming_timer:1.1,windup:1.1,attack_name:'Blast',pose:'windup'}];
  window.armingRun=run;document.querySelector('#rift-overlay').hidden=true;RiftRenderer.snapshot(run,true);
 },{run:data.run,enemy});
 await expect.poll(()=>page.evaluate(()=>armingLabels.includes('ARMING · HIT TO CANCEL · 1.1s')&&armingLabels.includes('INTENT · Arming blast')&&armingRings.length>0)).toBe(true);
 await page.locator('#rift-canvas').screenshot({path:testInfo.outputPath('arming-warning.png')});
 await page.evaluate(()=>{Object.assign(armingRun.enemies[0],{arming_timer:0,windup:0,attack_name:'',pose:'hit'});armingLabels=[];armingRings=[];armingRun.counter++;armingRun.events=[{id:armingRun.counter,kind:'arming_cancel',x:630,y:380,value:0}];RiftRenderer.snapshot(armingRun,true);});
 await expect.poll(()=>page.evaluate(()=>cancelledBlast)).toBe(true);
 await page.waitForTimeout(150);expect(await page.evaluate(()=>armingLabels)).toEqual([]);expect(await page.evaluate(()=>armingRings)).toEqual([]);
 await page.evaluate(()=>{Object.assign(armingRun.enemies[0],{arming_timer:1.1,windup:1.1,attack_name:'Blast',pose:'windup'});RiftDisplay.cleanScreenshot=true;armingLabels=[];armingRings=[];RiftRenderer.snapshot(armingRun,true);});
 await page.waitForTimeout(150);expect(await page.evaluate(()=>armingLabels)).toEqual([]);expect(await page.evaluate(()=>armingRings)).toEqual([]);
});
