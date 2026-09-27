const {test,expect}=require('@playwright/test');
test('boss fan warning draws five saved directions with reduced motion',async({page},testInfo)=>{
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
 const data=await(await page.request.get('/api/abyss/rift')).json();const boss=data.bestiary.find(e=>e.volley_fan);expect(boss).toBeTruthy();
 await page.evaluate(async({run,boss})=>{
  await RiftRenderer.ready;RiftRenderer.reduced=true;RiftDisplay.cameraSmooth=false;
  const ctx=document.querySelector('#rift-canvas').getContext('2d');
  window.fanLines=[];window.fanLabels=[];let start;
  const move=ctx.moveTo,line=ctx.lineTo,fill=ctx.fillText;
  ctx.moveTo=function(x,y){start=[x,y];return move.call(this,x,y);};
  ctx.lineTo=function(x,y){if(this.strokeStyle==='#b8e7ff'&&this.getLineDash().join(',')==='8,5')fanLines.push([x-start[0],y-start[1]]);return line.call(this,x,y);};
  ctx.fillText=function(text,...args){if(/FAN|JUMP OR MOVE/.test(text))fanLabels.push(text);return fill.call(this,text,...args);};
  run.status='fighting';run.paused=true;run.events=[];run.player.x=550;run.player.y=480;
  run.enemies=[{...boss,id:'fan',x:700,y:410,hp:1000,max_hp:1000,attacks:1,target_x:500,target_y:410,fan_rotation:-.18,attack_name:'Rotating Fan',windup:1.25,pose:'windup'}];
  window.fanRun=run;document.querySelector('#rift-overlay').hidden=true;RiftRenderer.snapshot(run,true);
 },{run:data.run,boss});
 await expect.poll(()=>page.evaluate(()=>fanLabels.includes('FAN · MOVE BETWEEN SHOTS'))).toBe(true);
 for(const rotation of [-.18,0,.18]){
  await page.evaluate(rotation=>{fanRun.enemies[0].fan_rotation=rotation;fanLines=[];fanLabels=[];RiftRenderer.snapshot(fanRun,true);},rotation);
  await expect.poll(()=>page.evaluate(()=>fanLines.length>=5)).toBe(true);
  const rays=await page.evaluate(()=>fanLines.slice(-5));
  for(let n=0;n<5;n++){const angle=Math.PI+rotation+(n-2)*.22;expect(rays[n][0]).toBeCloseTo(Math.cos(angle)*430,6);expect(rays[n][1]).toBeCloseTo(Math.sin(angle)*430,6);}
  expect(await page.evaluate(()=>fanLabels.some(s=>s.includes('JUMP OR MOVE')))).toBe(false);
 }
 await page.locator('#rift-canvas').screenshot({path:testInfo.outputPath('boss-fan.png')});
 await page.evaluate(()=>{RiftDisplay.cleanScreenshot=true;fanLines=[];fanLabels=[];RiftRenderer.snapshot(fanRun,true);});await page.waitForTimeout(150);
 expect(await page.evaluate(()=>fanLines)).toEqual([]);expect(await page.evaluate(()=>fanLabels)).toEqual([]);
});
