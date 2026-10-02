const {test,expect}=require('@playwright/test');
test('ultimate-saving objective follows confirmed keyboard casts and reloads',async({page})=>{
 await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();
 const goal=page.locator('#rift-objectives-list [data-objective-id="save_ultimate"]');await expect(goal).toContainText('Other abilities are allowed');
 await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();
 await page.keyboard.press('KeyR');
 const saved=async()=>(await(await page.request.get('/api/abyss/rift')).json()).run;
 await expect.poll(async()=>(await saved()).objectives.entries.find(e=>e.id==='save_ultimate').status).toBe('failed');
 await page.keyboard.press('Escape');await expect(page.locator('#rift-paused-badge')).toHaveText('Paused');await expect(goal).toContainText('1 ultimates cast');await expect(goal).toContainText('Cast an ultimate.');expect((await saved()).status).toBe('fighting');
 await page.reload();await expect(page.locator('#rift-start')).toBeEnabled();await expect(goal).toContainText('Failed');
});
