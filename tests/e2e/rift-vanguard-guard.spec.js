const {test,expect}=require('@playwright/test');
test('Vanguard primer and confirmed reward caption explain perfect-guard charge',async({page})=>{
 await page.goto('/abyss/rift?subclass=vanguard&scenario=spawn-hazards');await expect(page.locator('#rift-start')).toBeEnabled();await expect(page.locator('#rift-class-primer')).toContainText('Perfect-guard an enemy attack to gain one charge per guard raise (maximum three).');
 await page.locator('.rift-settings > summary').click();await page.locator('#rift-combat-captions').check();
 const {run}=await(await page.request.get('/api/abyss/rift')).json();await page.evaluate(run=>{run.paused=false;RiftFeedback.update(run,true,true);run.counter++;run.events=[{id:run.counter,kind:'vanguard_guard',x:run.player.x,y:run.player.y,value:0}];RiftFeedback.update(run,false,true);RiftFeedback.update(run,false,true);},run);
 await expect(page.locator('#rift-captions li')).toHaveText(['Vanguard perfect guard: +1 class charge']);
 await page.goto('/abyss/rift?subclass=marksman');await expect(page.locator('#rift-start')).toBeEnabled();await expect(page.locator('#rift-class-primer')).not.toContainText('Perfect-guard');
});
