const {test,expect}=require('@playwright/test');
test('contextual hint opt-out covers coaching, terrain, checkpoint and defeat help',async({page})=>{
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
 const show=async(status)=>page.evaluate(async status=>{const run=(await(await fetch('/api/abyss/rift')).json()).run;run.status=status;run.stats.empty_finishers=3;run.defeated_by_hazard={kind:'fire',jumpable:true};run.level.rooms[run.room].cover=[{x:run.player.x+10,y:run.player.y,w:20,h:20,material:'wood',hp:10,max_hp:10,id:'test'}];window.RiftHUD.update(run,false,true);},status);
 await show('cleared');await expect(page.locator('#rift-class-coaching')).toBeVisible();await expect(page.locator('#rift-checkpoint-guide')).toBeVisible();await expect(page.locator('#rift-terrain-hint')).toBeVisible();
 await page.locator('.rift-settings > summary').click();await page.locator('#rift-context-hints').uncheck();
 for(const id of ['rift-class-coaching','rift-checkpoint-guide','rift-terrain-hint','rift-walkthrough'])await expect(page.locator('#'+id)).toBeHidden();
 await show('cleared');await expect(page.locator('#rift-checkpoint-guide')).toBeHidden();await expect(page.locator('#rift-class-coaching')).toBeHidden();
 await show('defeated');await expect(page.locator('#rift-defeat-guide')).toBeHidden();await expect(page.locator('#rift-hazard-defeat-hint')).toBeHidden();
 await page.locator('#rift-walkthrough-review').click();await expect(page.locator('#rift-walkthrough')).toBeVisible();
 await page.reload();await expect(page.locator('#rift-start')).toBeEnabled();await expect(page.locator('#rift-context-hints')).not.toBeChecked();await expect(page.locator('#rift-walkthrough')).toBeHidden();await show('cleared');await expect(page.locator('#rift-class-coaching')).toBeHidden();
 await page.locator('.rift-settings > summary').click();await page.locator('#rift-context-hints').check();await show('cleared');await expect(page.locator('#rift-class-coaching')).toBeVisible();await expect(page.locator('#rift-checkpoint-guide')).toBeVisible();
});
