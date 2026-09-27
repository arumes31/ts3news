const {test,expect}=require('@playwright/test');
for(const reduced of [false,true])test(`boss shell progress and break reduced=${reduced}`,async({page},info)=>{
 const errors=[];page.on('pageerror',e=>errors.push(e.message));await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
 const data=await(await page.request.get('/api/abyss/rift')).json();
 await page.evaluate(async({run,bestiary,reduced})=>{
  await RiftRenderer.ready;RiftRenderer.reduced=reduced;RiftDisplay.enemyIntent=true;RiftDisplay.cameraSmooth=false;
  const ctx=document.querySelector('#rift-canvas').getContext('2d'),fill=ctx.fillText;window.shellLabels=[];
  ctx.fillText=function(text,...args){if(/SHELL|FAN/.test(text))shellLabels.push(text);return fill.call(this,text,...args);};
  const boss=bestiary.find(e=>e.volley_fan);if(boss.boss_shield_hp!==90)throw Error('canonical shell missing');
  run.status='fighting';run.paused=true;run.events=[];run.player.x=440;
  run.enemies=[{...boss,id:'kraken',x:650,y:410,hp:1000,max_hp:1000,attacks:1,windup:1.25,target_x:440,target_y:410,attack_name:'Rotating Fan',pose:'windup'}];
  window.shellRun=run;document.querySelector('#rift-overlay').hidden=true;RiftRenderer.snapshot(run,true);
 },{run:data.run,bestiary:data.bestiary,reduced});
 await expect.poll(()=>page.evaluate(()=>shellLabels.includes('SHELL 90/90 · BREAK WITH HITS')&&shellLabels.includes('FAN · MOVE BETWEEN SHOTS'))).toBe(true);
 await page.locator('#rift-canvas').screenshot({path:info.outputPath('shell-warning.png')});
 await page.evaluate(()=>{shellRun.enemies[0].boss_shield_hp=30;shellLabels=[];RiftRenderer.snapshot(shellRun,true);});await expect.poll(()=>page.evaluate(()=>shellLabels.includes('SHELL 30/90 · BREAK WITH HITS'))).toBe(true);
 await page.evaluate(()=>{shellRun.enemies[0].weak_point=.8;shellLabels=[];RiftRenderer.snapshot(shellRun,true);});await expect.poll(()=>page.evaluate(()=>shellLabels.includes('SHELL 30/90 · WEAK POINT BYPASSES'))).toBe(true);
 await page.evaluate(()=>{Object.assign(shellRun.enemies[0],{boss_shield_hp:0,windup:0,attack_name:'',pose:'stagger',weak_point:1.2});shellRun.counter++;shellRun.events=[{id:shellRun.counter,kind:'boss_shield_break',x:650,y:380,value:0}];shellLabels=[];RiftRenderer.snapshot(shellRun,true);});
 await expect.poll(()=>page.evaluate(()=>shellLabels.includes('SHELL BROKEN')&&shellLabels.includes('SHELL SHATTERED · WEAK POINT'))).toBe(true);
 await page.locator('#rift-canvas').screenshot({path:info.outputPath('shell-broken.png')});
 await page.evaluate(()=>{RiftDisplay.cleanScreenshot=true;shellLabels=[];RiftRenderer.snapshot(shellRun,true);});await page.waitForTimeout(120);expect(await page.evaluate(()=>shellLabels)).toEqual([]);expect(errors).toEqual([]);
});
