const {test,expect}=require('@playwright/test');
for(const outcome of ['cleared','defeated'])test('encounter result links unique canonical monsters: '+outcome,async({page})=>{
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
 const {run,bestiary}=await(await page.request.get('/api/abyss/rift')).json();
 const targets=[bestiary.find(e=>e.kind!=='boss'),bestiary.find(e=>e.kind==='boss')];
 run.last_encounter={mission:1,mission_name:'Ruins',room:0,room_name:'Gate',outcome,seconds:12,player_hp:100,player_max_hp:100,enemies:3,monster_keys:[targets[0].art_key,targets[1].art_key,targets[0].art_key,'monster:Removed catalog entry']};
 await page.evaluate(run=>window.RiftHUD.updateLastEncounter(run),run);
 const links=page.locator('#rift-result-monster-links button');await expect(links).toHaveCount(2);
 for(let i=0;i<targets.length;i++){
  await expect(links.nth(i)).toHaveText(targets[i].name);await links.nth(i).click();
  await expect(page.locator('#rift-monster-title')).toHaveText(targets[i].name);
  await expect(page.locator('#rift-monster-title')).toBeFocused();
  await page.evaluate(run=>window.RiftHUD.updateLastEncounter(run),run);
  await page.keyboard.press('Escape');await expect(links.nth(i)).toBeFocused();
 }
 await page.setViewportSize({width:390,height:1000});
 await links.first().scrollIntoViewIfNeeded();
 expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBeTruthy();
 delete run.last_encounter.monster_keys;
 await page.evaluate(run=>window.RiftHUD.updateLastEncounter(run),run);
 await expect(page.locator('#rift-result-monsters')).toBeHidden();await expect(links).toHaveCount(0);
});
