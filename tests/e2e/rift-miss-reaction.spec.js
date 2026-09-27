const {test,expect}=require('@playwright/test');
test('miss pressure cue renders with reduced motion and disappears on interruption',async({page},testInfo)=>{
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
 const data=await(await page.request.get('/api/abyss/rift')).json();
 await page.evaluate(async run=>{
  await RiftRenderer.ready;RiftRenderer.reduced=true;RiftDisplay.cameraSmooth=false;
  const ctx=document.querySelector('#rift-canvas').getContext('2d'),original=ctx.fillText;
  window.missLabels=[];ctx.fillText=function(text,...args){if(text.includes('MISSED STRIKE'))window.missLabels.push({text,fill:ctx.fillStyle});return original.call(this,text,...args);};
  run.status='fighting';run.paused=true;run.player.x=400;run.enemies=[{id:'pressure',name:'Pressing fighter',kind:'goblin',x:250,y:410,hp:100,max_hp:100,facing:1,pose:'run',react_miss_timer:.22}];
  window.missRun=run;document.querySelector('#rift-overlay').hidden=true;RiftRenderer.snapshot(run,true);
 },data.run);
 await expect.poll(()=>page.evaluate(()=>missLabels.some(label=>label.text==='PRESSING · MISSED STRIKE'&&label.fill==='#ffe3b0'))).toBe(true);
 await page.locator('#rift-canvas').screenshot({path:testInfo.outputPath('miss-pressure.png')});
 await page.evaluate(()=>{missRun.enemies[0].react_miss_timer=0;missRun.enemies[0].pose='hit';RiftRenderer.snapshot(missRun,true);});
 await page.waitForTimeout(100);await page.evaluate(()=>{missLabels=[];});await page.waitForTimeout(100);
 expect(await page.evaluate(()=>missLabels)).toEqual([]);
});
