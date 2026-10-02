const {test,expect}=require('@playwright/test');
test('optional intent overlay reports combat states, persists, and respects clean screenshots',async({page},testInfo)=>{
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
 const toggle=page.locator('#rift-enemy-intent');await expect(toggle).not.toBeChecked();
 await page.locator('.rift-settings').evaluate(el=>el.open=true);await toggle.check();
 await page.reload();await expect(toggle).toBeChecked();
 const data=await(await page.request.get('/api/abyss/rift')).json();
 const cases=[
  [{kind:'archer',pose:'run'},'Moving'],[{kind:'goblin',pose:'run'},'Approaching'],[{kind:'knight',guard:true},'Guarding'],
  [{kind:'archer',windup:.5},'Aiming'],[{kind:'boss',windup:.5},'Preparing attack'],
  [{kind:'goblin',charging:true,attack_name:'Charge',windup:.5},'Preparing charge'],
  [{kind:'goblin',charge_active:true},'Charging'],[{kind:'goblin',charge_recovery:.5},'Recovering'],
  [{kind:'goblin',react_miss_timer:.2},'Pressing'],[{kind:'treasure',fleeing:true},'Fleeing'],
  [{kind:'goblin',patrol:true,alerted:false},'Patrolling'],[{kind:'goblin',awareness_timer:.2},'Alerting'],
  [{kind:'archer',reposition_timer:.5},'Repositioning'],[{kind:'goblin',pose:'attack',pose_time:.2},'Attacking'],
  [{kind:'goblin',pose:'hit',pose_time:.2},'Hit reaction'],[{kind:'goblin',pose:'spawn',arrival_vulnerability:.5},'Arriving'],[{kind:'goblin',cooldown:.4},'Recovering'],[{kind:'goblin',pose:'stagger',pose_time:.2},'Stunned'],[{kind:'goblin',knockdown:.5},'Getting up'],
  [{kind:'goblin',pose:'idle'},'Holding'],[{kind:'goblin',hp:0},null],
  [{kind:'goblin',id:'practice-target'},null],[{kind:'totem'},null]
 ];
 await page.evaluate(async run=>{
  await RiftRenderer.ready;RiftRenderer.reduced=true;RiftDisplay.cameraSmooth=false;RiftDisplay.enemyNames='none';
  const ctx=document.querySelector('#rift-canvas').getContext('2d'),original=ctx.fillText;
  window.intentLabels=[];ctx.fillText=function(text,...args){if(text.startsWith('INTENT ·'))intentLabels.push({text,fill:ctx.fillStyle});return original.call(this,text,...args);};
  run.status='fighting';run.paused=true;run.player.x=500;window.intentRun=run;document.querySelector('#rift-overlay').hidden=true;
 },data.run);
 for(const [state,label] of cases){
  await page.evaluate(state=>{intentLabels=[];intentRun.enemies=[{id:'intent-probe',x:420,y:410,hp:100,max_hp:100,pose:'idle',facing:1,...state}];RiftRenderer.snapshot(intentRun,true);},state);
  if(label) await expect.poll(()=>page.evaluate(()=>intentLabels.at(-1)?.text)).toBe('INTENT · '+label);
  else {await page.waitForTimeout(80);expect(await page.evaluate(()=>intentLabels)).toEqual([]);}
 }
 await page.evaluate(()=>{intentRun.enemies=[{id:'intent-probe',kind:'knight',x:420,y:410,hp:100,max_hp:100,pose:'guard',guard:true}];RiftRenderer.snapshot(intentRun,true);});
 await expect.poll(()=>page.evaluate(()=>intentLabels.at(-1)?.text)).toBe('INTENT · Guarding');
 expect(await page.evaluate(()=>intentLabels.at(-1).fill)).toBe('#b7f2ff');
 await page.locator('#rift-canvas').screenshot({path:testInfo.outputPath('enemy-intent.png')});
 await page.evaluate(()=>{RiftDisplay.cleanScreenshot=true;intentLabels=[];});await page.waitForTimeout(100);expect(await page.evaluate(()=>intentLabels)).toEqual([]);
 await page.evaluate(()=>{RiftDisplay.cleanScreenshot=false;});await page.locator('.rift-settings').evaluate(el=>el.open=true);await toggle.uncheck();
 await page.evaluate(()=>{intentLabels=[];});await page.waitForTimeout(100);expect(await page.evaluate(()=>intentLabels)).toEqual([]);
 await page.reload();await expect(toggle).not.toBeChecked();
 await page.locator('.rift-settings').evaluate(el=>el.open=true);await toggle.check();await page.locator('#rift-display-preset').selectOption('balanced');await page.locator('#rift-apply-preset').click();await expect(toggle).not.toBeChecked();
});
