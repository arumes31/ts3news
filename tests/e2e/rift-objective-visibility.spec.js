const {test,expect}=require('@playwright/test');
test('hide objectives without stopping tracking and restore latest progress after reload',async({page})=>{
 await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('.rift-settings > summary').click();
 const toggle=page.locator('#rift-show-objectives');await expect(toggle).toBeVisible();await toggle.uncheck();await expect(page.locator('#rift-optional-objectives')).toBeHidden();
 await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();await page.keyboard.press('Digit1');
 const saved=async()=>(await(await page.request.get('/api/abyss/rift')).json()).run;
 await expect.poll(async()=>(await saved()).objectives.entries.find(e=>e.id==='basic_only').status).toBe('failed');await page.keyboard.press('Escape');await expect(page.locator('#rift-paused-badge')).toHaveText('Paused');
 await page.reload();await expect(page.locator('#rift-start')).toBeEnabled();await expect(page.locator('#rift-optional-objectives')).toBeHidden();await page.locator('.rift-settings > summary').click();await expect(toggle).not.toBeChecked();
 const before=await saved();await toggle.check();await expect(page.locator('#rift-optional-objectives')).toBeVisible();await expect(page.locator('#rift-objectives-list [data-objective-id="basic_only"]')).toContainText('Used an ability.');expect(await saved()).toEqual(before);
});
test('objective visibility remains usable when storage is blocked',async({page})=>{
 await page.addInitScript(()=>{Storage.prototype.getItem=()=>{throw Error('blocked')};Storage.prototype.setItem=()=>{throw Error('blocked')};});
 await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('.rift-settings > summary').click();
 const toggle=page.locator('#rift-show-objectives');await toggle.uncheck();await expect(page.locator('#rift-optional-objectives')).toBeHidden();await toggle.check();await expect(page.locator('#rift-optional-objectives')).toBeVisible();
});
