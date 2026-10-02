const {test,expect}=require('@playwright/test');
for(const reduced of [false,true])test(`boss summon intermission and dismissal reduced=${reduced}`,async({page},info)=>{
 const errors=[];page.on('pageerror',e=>errors.push(e.message));await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
 const data=await(await page.request.get('/api/abyss/rift')).json();
 await page.evaluate(async({run,bestiary,reduced})=>{
  await RiftRenderer.ready;RiftRenderer.reduced=reduced;RiftDisplay.enemyIntent=true;RiftDisplay.cameraSmooth=false;
  const ctx=document.querySelector('#rift-canvas').getContext('2d'),fill=ctx.fillText;window.summonLabels=[];
  ctx.fillText=function(text,...args){if(/SUMMON|INTERMISSION|Summoning/.test(text))summonLabels.push(text);return fill.call(this,text,...args);};
  const boss=bestiary.find(e=>e.ring_attack),minion=bestiary.find(e=>e.kind==='goblin');
  run.status='fighting';run.paused=true;run.events=[];run.player.x=300;
  run.enemies=[{...boss,id:'boss',x:650,y:410,hp:1000,max_hp:1000,summon_timer:4,summon_released:false,pose:'idle'}, {...minion,id:'minion',x:490,y:430,hp:0,max_hp:100,summon_owner:'boss',summoned:true,arrival_vulnerability:.85,pose:'spawn'}];
  window.summonRun=run;document.querySelector('#rift-overlay').hidden=true;RiftRenderer.snapshot(run,true);
 },{run:data.run,bestiary:data.bestiary,reduced});
 await expect.poll(()=>page.evaluate(()=>summonLabels.includes('SUMMONS IN 1.0s · 0/2 ALLIES')&&summonLabels.includes('INTENT · Summoning allies'))).toBe(true);
 await page.locator('#rift-canvas').screenshot({path:info.outputPath('summon-warning.png')});
 await page.evaluate(()=>{Object.assign(summonRun.enemies[0],{summon_timer:2.9,summon_released:true});summonRun.enemies[1].hp=100;summonLabels=[];RiftRenderer.snapshot(summonRun,true);});
 await expect.poll(()=>page.evaluate(()=>summonLabels.includes('MINION INTERMISSION · 1/2 ALLIES')&&summonLabels.includes('SUMMON · ARRIVAL VULNERABLE'))).toBe(true);
 await page.locator('#rift-canvas').screenshot({path:info.outputPath('summon-arrival.png')});
 await page.evaluate(()=>{summonRun.enemies.forEach(e=>e.hp=0);summonRun.counter++;summonRun.events=[{id:summonRun.counter,kind:'summon_dismiss',x:490,y:430,value:0}];summonLabels=[];RiftRenderer.snapshot(summonRun,true);});
 await expect.poll(()=>page.evaluate(()=>summonLabels.includes('SUMMON DISMISSED'))).toBe(true);
 await page.evaluate(()=>{RiftDisplay.cleanScreenshot=true;summonLabels=[];RiftRenderer.snapshot(summonRun,true);});await page.waitForTimeout(120);expect(await page.evaluate(()=>summonLabels)).toEqual([]);expect(errors).toEqual([]);
});
