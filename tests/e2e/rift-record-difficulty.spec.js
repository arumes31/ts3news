const {test,expect}=require('@playwright/test');
test('attempts retain difficulty and compare only matching difficulty',async({page})=>{
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();const saved=(await(await page.request.get('/api/abyss/rift')).json()).run;
 const show=async(difficulty)=>page.evaluate(({run,difficulty})=>{const record={mission:1,outcome:'completed',at_ms:1700000000000,class:'vanguard',seconds:10,hp:50,max_hp:100,splits:[3,3,4]};run.attempt_history=[{...record,difficulty:'Wayfarer'},{...record,seconds:12,difficulty}];RiftRecords.update(run);},{run:saved,difficulty});
 await show('Veteran');await expect(page.locator('#rift-attempt-list')).toContainText('Wayfarer');await expect(page.locator('#rift-attempt-list')).toContainText('Veteran');await expect(page.locator('#rift-attempt-comparison')).toContainText('No previous recorded attempt');
 await show('Wayfarer');await expect(page.locator('#rift-attempt-comparison')).toContainText('combat time +2.0s');await show(undefined);await expect(page.locator('#rift-attempt-list')).toContainText('Difficulty unavailable');await expect(page.locator('#rift-attempt-comparison')).toContainText('unknown difficulty');expect((await(await page.request.get('/api/abyss/rift')).json()).run).toEqual(saved);
});
