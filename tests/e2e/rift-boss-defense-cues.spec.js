const {test,expect}=require('@playwright/test');
test('boss warnings distinguish slam evasion from directional volley guarding',async({page})=>{
 await page.goto('/abyss/rift?scenario=boss-windup');await expect(page.locator('#rift-start')).toBeEnabled();
 const run=(await(await page.request.get('/api/abyss/rift')).json()).run;
 for(const [attacks,name,cue] of [[0,'Ground Slam','Jump or move clear'],[1,'Aimed Volley','Move to evade; face the shot to guard']]){
  await page.evaluate(({run,attacks,name})=>{document.getElementById('rift-overlay').hidden=true;run.paused=false;run.status='fighting';const boss=run.enemies.find(e=>e.kind==='boss');boss.art_key='monster:Test';boss.attacks=attacks;boss.windup=.5;boss.attack_name=name;window.RiftHUD.update(run,true);},{run,attacks,name});
  await expect(page.locator('#rift-boss-attack')).toContainText(cue);
  await expect(page.locator('#rift-announcer')).toContainText(cue);
 }
});
