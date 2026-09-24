const {test,expect}=require('@playwright/test');
test('hazard coaching waits for repeated damage, expires, and honors saved preference',async({page})=>{
 await page.goto('/abyss/rift?scenario=spawn-hazards');await expect(page.locator('#rift-start')).toBeEnabled();const {run}=await(await page.request.get('/api/abyss/rift')).json();
 const show=async(damage,seconds,replay=false)=>page.evaluate(({run,damage,seconds,replay})=>{run.stats.hazard_damage_taken=damage;run.stats.seconds=seconds;window.RiftHUD.update(run,false,replay);},{run,damage,seconds,replay});
 const hint=page.locator('#rift-hazard-coaching');
 await show(100,10,true);await expect(hint).toBeHidden();
 await show(110,11);await show(120,12);await expect(hint).toBeHidden();
 await show(130,13);await expect(hint).toContainText('Repeated hazard damage');await expect(hint).toBeVisible();
 await show(140,22);await expect(hint).toBeHidden();await show(150,23);await show(160,24);await expect(hint).toBeHidden();
 await page.locator('.rift-settings > summary').click();await page.locator('#rift-context-hints').uncheck();
 await show(0,0,true);await show(10,1);await show(20,2);await show(30,3);await expect(hint).toBeHidden();
 await page.evaluate(()=>history.replaceState(null,'','/abyss/rift'));await page.reload();await expect(page.locator('#rift-start')).toBeEnabled();await expect(page.locator('#rift-context-hints')).not.toBeChecked();
});
