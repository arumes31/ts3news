const {test,expect}=require('@playwright/test');
test('support retreat shows readable cover and intent cues',async({page},testInfo)=>{
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
 const data=await(await page.request.get('/api/abyss/rift')).json(),support=data.bestiary.find(e=>e.support);expect(support).toBeTruthy();
 await page.evaluate(async({run,support})=>{
  await RiftRenderer.ready;RiftRenderer.reduced=true;RiftDisplay.cameraSmooth=false;RiftDisplay.enemyIntent=true;
  const ctx=document.querySelector('#rift-canvas').getContext('2d'),original=ctx.fillText;
  window.coverLabels=[];ctx.fillText=function(text,...args){if(text==='SEEKING COVER'||text==='INTENT · Seeking defender')coverLabels.push({text,fill:ctx.fillStyle});return original.call(this,text,...args);};
  run.status='fighting';run.paused=true;run.player.x=500;run.enemies=[{...support,id:'support',x:630,y:410,hp:100,max_hp:100,support_cover_id:'defender',pose:'run',facing:-1},{id:'defender',kind:'knight',x:760,y:440,hp:100,max_hp:100,pose:'guard',guard:true}];
  window.coverRun=run;document.querySelector('#rift-overlay').hidden=true;RiftRenderer.snapshot(run,true);
 },{run:data.run,support});
 await expect.poll(()=>page.evaluate(()=>coverLabels.some(label=>label.text==='SEEKING COVER'&&label.fill==='#b7dcff'))).toBe(true);
 await expect.poll(()=>page.evaluate(()=>coverLabels.some(label=>label.text==='INTENT · Seeking defender'))).toBe(true);
 await page.locator('#rift-canvas').screenshot({path:testInfo.outputPath('support-retreat.png')});
 await page.evaluate(()=>{Object.assign(coverRun.enemies[0],{support_cover_id:'',pose:'idle'});RiftRenderer.snapshot(coverRun,true);coverLabels=[];});
 await page.waitForTimeout(100);expect(await page.evaluate(()=>coverLabels)).toEqual([]);
});
