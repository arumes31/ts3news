const {test,expect}=require('@playwright/test');
test('enemy mend marks the ally and shows interrupt guidance',async({page},testInfo)=>{
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
 const data=await(await page.request.get('/api/abyss/rift')).json(),healer=data.bestiary.find(e=>e.healer);expect(healer).toBeTruthy();
 await page.evaluate(async({run,healer})=>{
  await RiftRenderer.ready;RiftRenderer.reduced=true;RiftDisplay.cameraSmooth=false;RiftDisplay.enemyIntent=true;
  const ctx=document.querySelector('#rift-canvas').getContext('2d'),original=ctx.fillText;
  window.mendLabels=[];ctx.fillText=function(text,...args){if(text.includes('MEND')||text.includes('Mending'))mendLabels.push(text);return original.call(this,text,...args);};
  run.status='fighting';run.paused=true;run.player.x=500;run.enemies=[{...healer,id:'healer',x:630,y:410,hp:100,max_hp:100,heal_target:'ally',attack_name:'Mend ally',windup:.8,pose:'cast'},{id:'ally',kind:'knight',name:'Wounded guard',x:820,y:440,hp:20,max_hp:100,pose:'idle'}];
  window.mendRun=run;document.querySelector('#rift-overlay').hidden=true;RiftRenderer.snapshot(run,true);
 },{run:data.run,healer});
 await expect.poll(()=>page.evaluate(()=>mendLabels.includes('MEND · INTERRUPT'))).toBe(true);
 await expect.poll(()=>page.evaluate(()=>mendLabels.includes('INTENT · Mending ally'))).toBe(true);
 await page.locator('#rift-canvas').screenshot({path:testInfo.outputPath('enemy-mend.png')});
 await page.evaluate(()=>{Object.assign(mendRun.enemies[0],{heal_target:'',attack_name:'',windup:0,pose:'hit'});RiftRenderer.snapshot(mendRun,true);mendLabels=[];});
 await page.waitForTimeout(100);expect(await page.evaluate(()=>mendLabels)).toEqual([]);
});
