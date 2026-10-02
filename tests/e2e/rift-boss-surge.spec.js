const {test,expect}=require('@playwright/test');
for(const reduced of [false,true])test(`arena surge warns jump instead of guard reduced=${reduced}`,async({page},info)=>{
 const errors=[];page.on('pageerror',e=>errors.push(e.message));await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
 const data=await(await page.request.get('/api/abyss/rift')).json();
 await page.evaluate(async({run,bestiary,reduced})=>{
  await RiftRenderer.ready;RiftRenderer.reduced=reduced;RiftDisplay.enemyIntent=true;RiftDisplay.cameraSmooth=false;RiftDisplay.hazardContrast=false;
  const ctx=document.querySelector('#rift-canvas').getContext('2d'),fill=ctx.fillText,rect=ctx.strokeRect;window.surgeLabels=[];window.surgeRects=[];
  ctx.fillText=function(text,...args){if(/SURGE|GUARD WON|arena surge/.test(text))surgeLabels.push(text);return fill.call(this,text,...args);};
  ctx.strokeRect=function(...args){if(args[2]===1530&&args[3]===175&&this.strokeStyle==='#ffd7a1')surgeRects.push(args);return rect.apply(this,args);};
  run.status='fighting';run.paused=true;run.events=[];run.player.x=440;
  run.enemies=[{...bestiary.find(e=>e.kind==='boss'&&e.charging),id:'dragon',x:650,y:410,hp:1000,max_hp:1000,attacks:7,windup:2,attack_name:'Ground Surge',pose:'windup'}];
  window.surgeRun=run;document.querySelector('#rift-overlay').hidden=true;RiftRenderer.snapshot(run,true);
 },{run:data.run,bestiary:data.bestiary,reduced});
 await expect.poll(()=>page.evaluate(()=>surgeLabels.includes('ARENA-WIDE SURGE · 2.0s')&&surgeLabels.includes('JUMP NEAR IMPACT · GUARD WON’T STOP IT')&&surgeLabels.includes('INTENT · Preparing arena surge')&&surgeRects.length>0)).toBe(true);
 await page.locator('#rift-canvas').screenshot({path:info.outputPath('surge-warning.png')});
 await page.evaluate(()=>{Object.assign(surgeRun.enemies[0],{windup:0,attack_name:'',attacks:8,pose:'idle'});surgeLabels=[];surgeRects=[];RiftRenderer.snapshot(surgeRun,true);});await page.waitForTimeout(120);expect(await page.evaluate(()=>[surgeLabels.length,surgeRects.length])).toEqual([0,0]);
 await page.evaluate(()=>{Object.assign(surgeRun.enemies[0],{windup:2,attack_name:'Ground Surge'});RiftDisplay.cleanScreenshot=true;surgeLabels=[];surgeRects=[];RiftRenderer.snapshot(surgeRun,true);});await page.waitForTimeout(120);expect(await page.evaluate(()=>[surgeLabels.length,surgeRects.length])).toEqual([0,0]);expect(errors).toEqual([]);
});
