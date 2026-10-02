const {test,expect}=require('@playwright/test');
test('boss stagger meter and recovery are distinct from health and readable with reduced motion',async({page},testInfo)=>{
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
 const data=await(await page.request.get('/api/abyss/rift')).json();
 const boss=data.bestiary.find(e=>e.kind==='boss');expect(boss).toBeTruthy();
 await page.evaluate(async({run,boss})=>{
  await RiftRenderer.ready;RiftRenderer.reduced=true;RiftDisplay.enemyIntent=true;
  const ctx=document.querySelector('#rift-canvas').getContext('2d'),fill=ctx.fillText;
  window.breakLabels=[];ctx.fillText=function(text,...args){if(/STAGGER|GUARD BROKEN|Guard broken/.test(text))breakLabels.push(text);return fill.call(this,text,...args);};
  run.status='fighting';run.paused=true;run.player.x=400;run.events=[];
  run.enemies=[{...boss,id:'boss',x:500,y:410,hp:900,max_hp:1000,boss_stagger:80,pose:'idle',windup:0}];
  window.breakRun=run;document.querySelector('#rift-overlay').hidden=true;RiftRenderer.snapshot(run,true);
 },{run:data.run,boss});
 await expect.poll(()=>page.evaluate(()=>breakLabels.includes('STAGGER 80/100'))).toBe(true);
 await page.locator('#rift-canvas').screenshot({path:testInfo.outputPath('boss-stagger.png')});
 await page.evaluate(()=>{Object.assign(breakRun.enemies[0],{boss_stagger:0,boss_guard_break:1.5,boss_stagger_grace:3.5,pose:'stagger',pose_time:1.5});breakLabels=[];RiftRenderer.snapshot(breakRun,true);});
 await expect.poll(()=>page.evaluate(()=>breakLabels.includes('GUARD BROKEN · 1.5s')&&breakLabels.includes('INTENT · Guard broken'))).toBe(true);
 await page.locator('#rift-canvas').screenshot({path:testInfo.outputPath('boss-guard-break.png')});
 await page.evaluate(()=>{Object.assign(breakRun.enemies[0],{boss_guard_break:0,boss_stagger_grace:2,pose:'idle',pose_time:0});breakLabels=[];RiftRenderer.snapshot(breakRun,true);});
 await expect.poll(()=>page.evaluate(()=>breakLabels.includes('STAGGER RECOVERING · 2.0s'))).toBe(true);
 await page.evaluate(()=>{RiftDisplay.cleanScreenshot=true;breakLabels=[];RiftRenderer.snapshot(breakRun,true);});
 await page.waitForTimeout(150);expect(await page.evaluate(()=>breakLabels)).toEqual([]);
});
