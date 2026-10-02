const {test,expect}=require('@playwright/test');
test('slow status names its saved source and clears with the timer',async({page})=>{
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();const {run}=await(await page.request.get('/api/abyss/rift')).json();
 for(const [source,expected] of [['ice','Slowed by Ice'],['poison','Slowed by Poison'],['thorns','Slowed by Thorns'],[undefined,'Slowed']]){
  const area=await page.evaluate(({run,source})=>{run.skill_timers.slowed=1.2;run.slow_source=source;run.level.rooms[run.room].hazards=[];run.enemies=[];RiftHUD.update(run);return RiftHUD.detectPlayerAreaEffects(run);},{run,source});
  await expect(page.locator('#rift-slow-state')).toHaveText(expected+' 1.2s');expect(area.label).toBe('Area effect: '+expected+' (1.2s)');
 }
 await page.evaluate(run=>{run.skill_timers.slowed=0;run.slow_source='ice';RiftHUD.update(run);},run);await expect(page.locator('#rift-slow-state')).toHaveText('Normal speed');
 await page.setViewportSize({width:390,height:844});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
});
