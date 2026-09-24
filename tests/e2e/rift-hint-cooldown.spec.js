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

test('class coaching waits for hazard advice and expires without repeating',async({page})=>{
 await page.goto('/abyss/rift?scenario=spawn-hazards');await expect(page.locator('#rift-start')).toBeEnabled();const {run}=await(await page.request.get('/api/abyss/rift')).json();
 const show=async(count,seconds)=>page.evaluate(({run,count,seconds})=>{run.stats.empty_finishers=count;run.stats.hazard_damage_taken=count*10;run.stats.seconds=seconds;RiftHUD.update(run,false);},{run,count,seconds});
 const coaching=page.locator('#rift-class-coaching');
 await show(0,0);await show(1,1);await show(2,2);await show(3,3);await expect(page.locator('#rift-hazard-coaching')).toBeVisible();await expect(coaching).toBeHidden();
 await show(3,22);await expect(coaching).toBeHidden();await show(3,23);await expect(coaching).toBeVisible();await show(3,32);await expect(coaching).toBeHidden();await show(6,60);await expect(coaching).toBeHidden();
});

test('terrain coaching limits new prompts and hides advice for cover left behind',async({page})=>{
 await page.goto('/abyss/rift?scenario=spawn-hazards');await expect(page.locator('#rift-start')).toBeEnabled();const {run}=await(await page.request.get('/api/abyss/rift')).json();
 const show=async(kind,seconds)=>page.evaluate(({run,kind,seconds})=>{run.stats.seconds=seconds;const arena=run.level.rooms[run.room];arena.obstacles=[];arena.high_cover=[];arena.drop_edges=[];arena.cover=kind?[{id:'cover',material:kind,x:run.player.x+10,y:run.player.y,w:20,h:20,hp:100,max_hp:100}]:[];RiftHUD.update(run,false);},{run,kind,seconds});
 const hint=page.locator('#rift-terrain-hint');
 await show('wood',0);await expect(hint).toBeVisible();await show('stone',1);await expect(hint).toBeHidden();await show('stone',19);await expect(hint).toBeHidden();await show('stone',20);await expect(hint).toBeVisible();await expect(hint).toContainText('Stone cover');
 await show(null,21);await expect(hint).toBeHidden();await show('stone',22);await expect(hint).toBeVisible();await show('stone',28);await expect(hint).toBeHidden();await show('stone',60);await expect(hint).toBeHidden();
});
