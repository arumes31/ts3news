const {test,expect}=require('@playwright/test');
test('wounded elite cue is readable and distinct from optional intent',async({page},testInfo)=>{
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
 const data=await(await page.request.get('/api/abyss/rift')).json();
 const elite=data.bestiary.find(e=>e.elite&&e.kind==='knight');expect(elite).toBeTruthy();
 await page.evaluate(async({run,elite})=>{
  await RiftRenderer.ready;RiftRenderer.reduced=true;RiftDisplay.cameraSmooth=false;RiftDisplay.enemyIntent=true;
  const ctx=document.querySelector('#rift-canvas').getContext('2d'),original=ctx.fillText;
  window.eliteLabels=[];ctx.fillText=function(text,...args){if(text==='LAST STAND'||text.startsWith('INTENT ·'))eliteLabels.push({text,fill:ctx.fillStyle});return original.call(this,text,...args);};
  run.status='fighting';run.paused=true;run.player.x=500;run.enemies=[{...elite,id:'elite',x:650,y:410,hp:35,max_hp:100,elite:true,enraged:true,charging:true,guard:true,pose:'guard'}];
  window.eliteRun=run;document.querySelector('#rift-overlay').hidden=true;RiftRenderer.snapshot(run,true);
 },{run:data.run,elite});
 await expect.poll(()=>page.evaluate(()=>eliteLabels.some(label=>label.text==='LAST STAND'&&label.fill==='#ffd18a'))).toBe(true);
 await expect.poll(()=>page.evaluate(()=>eliteLabels.some(label=>label.text==='INTENT · Guarding'))).toBe(true);
 await page.locator('#rift-canvas').screenshot({path:testInfo.outputPath('wounded-elite.png')});
 for(const patch of [{enraged:false},{enraged:true,hp:0}]){
  await page.evaluate(patch=>{Object.assign(eliteRun.enemies[0],patch);RiftRenderer.snapshot(eliteRun,true);eliteLabels=[];},patch);
  await page.waitForTimeout(100);expect(await page.evaluate(()=>eliteLabels.some(label=>label.text==='LAST STAND'))).toBe(false);
 }
});
