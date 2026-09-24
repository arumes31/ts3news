const {test,expect}=require('@playwright/test');
test('miss and hazard coaching share a combat-time cooldown across rooms',async({page})=>{
 await page.goto('/abyss/rift?scenario=spawn-hazards');await expect(page.locator('#rift-start')).toBeEnabled();const {run}=await(await page.request.get('/api/abyss/rift')).json();
 const show=async(count,seconds,room=run.room,replay=false)=>page.evaluate(({run,count,seconds,room,replay})=>{run.room=room;run.stats.basic_misses=count;run.stats.hazard_damage_taken=count*10;run.stats.seconds=seconds;RiftHUD.update(run,false,replay);},{run,count,seconds,room,replay});
 const hazard=page.locator('#rift-hazard-coaching'),miss=page.locator('#rift-miss-coaching');
 await show(0,0,run.room,true);await show(1,1);await show(2,2);await show(3,3);
 await expect(hazard).toBeVisible();await expect(miss).toBeHidden();
 await show(3,12);await expect(hazard).toBeHidden();await expect(miss).toBeHidden();
 await show(3,22);await expect(miss).toBeHidden();await show(3,23);await expect(miss).toBeVisible();
 await show(3,24,0);await expect(miss).toBeHidden();await show(4,25,0);await show(5,26,0);await show(6,27,0);await expect(hazard).toBeHidden();await expect(miss).toBeHidden();
 await show(6,43,0);await expect(hazard).toBeVisible();await expect(miss).toBeHidden();
});
