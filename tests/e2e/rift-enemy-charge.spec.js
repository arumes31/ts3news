const {test,expect}=require('@playwright/test');
for(const boss of [false,true])test((boss?'boss: ':'')+'charge path and recovery cues remain visible with reduced motion',async({page},testInfo)=>{
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
 const data=await(await page.request.get('/api/abyss/rift')).json();
 const charger=data.bestiary.find(e=>e.charging&&(e.kind==='boss')===boss);expect(charger).toBeTruthy();
 await page.evaluate(async({run,charger,boss})=>{
  await RiftRenderer.ready;RiftRenderer.reduced=true;RiftDisplay.cameraSmooth=false;
  const ctx=document.querySelector('#rift-canvas').getContext('2d'),original=ctx.fillText;
  window.chargeLabels=[];ctx.fillText=function(text,...args){if(/CHARGE|RECOVERING|JUMP OR MOVE/.test(text))window.chargeLabels.push(text);return original.call(this,text,...args);};
  run.status='fighting';run.paused=true;run.player.x=400;run.player.y=470;run.enemies=[{...charger,id:'charger',x:220,y:410,hp:100,max_hp:100,facing:1,charging:true,attacks:2,attack_name:'Charge',windup:boss?1.15:.7,target_x:470,target_y:410,pose:'windup'}];
  window.chargeRun=run;window.chargeBoss=boss;document.querySelector('#rift-overlay').hidden=true;RiftRenderer.snapshot(run,true);
 },{run:data.run,charger,boss});
 await expect.poll(()=>page.evaluate(()=>window.chargeLabels.includes('CHARGE · SIDESTEP'))).toBe(true);
 expect(await page.evaluate(()=>chargeLabels.some(s=>s.includes('JUMP OR MOVE')))).toBe(false);
 await page.locator('#rift-canvas').screenshot({path:testInfo.outputPath('charge-warning.png')});
 await page.evaluate(()=>{const e=chargeRun.enemies[0];Object.assign(e,{attack_name:'',windup:0,charge_recovery:chargeBoss?1.2:.85,pose:'recovery'});RiftRenderer.snapshot(chargeRun,true);});
 await expect.poll(()=>page.evaluate(()=>window.chargeLabels.some(s=>s.startsWith(chargeBoss?'RECOVERING · 1.2':'RECOVERING · 0.8')))).toBe(true);
 await page.locator('#rift-canvas').screenshot({path:testInfo.outputPath('charge-recovery.png')});
});
