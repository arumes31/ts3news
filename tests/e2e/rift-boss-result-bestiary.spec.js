const {test,expect}=require('@playwright/test');
for(const outcome of ['cleared','defeated'])test('boss result opens the matching bestiary and restores focus: '+outcome,async({page})=>{
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
 const {run,bestiary}=await(await page.request.get('/api/abyss/rift')).json();const bosses=bestiary.filter(e=>e.kind==='boss'),target=bosses[outcome==='defeated'?1:0];
 await page.evaluate(({run,target,first,outcome})=>{run.last_encounter={mission:1,mission_name:'Ruins',room:0,room_name:'Gate',outcome,seconds:12,player_hp:outcome==='defeated'?0:100,player_max_hp:100,enemies:1,boss_encounter:true,boss_name:first.name,defeated_by_boss:outcome==='defeated'?target.name:''};window.RiftHUD.updateLastEncounter(run);},{run,target,first:bosses[0],outcome});
 const link=page.locator('#rift-result-bestiary');await expect(link).toBeVisible();await link.click();
 await expect(page.locator('#rift-monster-title')).toHaveText(target.name);await expect(page.locator('#rift-monster-title')).toBeFocused();
 await page.keyboard.press('Escape');await expect(link).toBeFocused();
 await page.evaluate(run=>{run.last_encounter={mission:1,mission_name:'Ruins',room:0,room_name:'Gate',outcome:'cleared',seconds:1,player_hp:100,player_max_hp:100,boss_name:'Removed catalog boss'};window.RiftHUD.updateLastEncounter(run);},run);
 await expect(link).toBeHidden();
});
