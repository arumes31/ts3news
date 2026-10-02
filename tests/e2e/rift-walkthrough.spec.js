const {test,expect}=require('@playwright/test');

test('first-visit walkthrough can be skipped immediately and reopened after reload',async({page})=>{
 await page.goto('/abyss/rift');await expect(page.locator('#rift-walkthrough')).toBeVisible();await expect(page.locator('#rift-walkthrough-title')).toHaveText('Move and jump');await page.locator('#rift-walkthrough-skip').click();await expect(page.locator('#rift-walkthrough')).toBeHidden();await page.reload();await expect(page.locator('#rift-start')).toBeEnabled();await expect(page.locator('#rift-walkthrough')).toBeHidden();
 await page.setViewportSize({width:390,height:844});await page.locator('.rift-settings > summary').click();await page.locator('#rift-walkthrough-review').click();expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);await expect(page.locator('#rift-walkthrough')).toBeVisible();await expect(page.locator('#rift-walkthrough-title')).toHaveText('Move and jump');
});

test('walkthrough follows bindings and remembers completion without starting an expedition',async({page})=>{
 await page.addInitScript(()=>localStorage.setItem('riftBindings',JSON.stringify({version:1,bindings:{jump:['KeyF']}})));
 await page.goto('/abyss/rift');await expect(page.locator('#rift-walkthrough-copy')).toContainText('Jump with F');await expect(page.locator('#rift-walkthrough-back')).toBeDisabled();
 for(const title of ['Fight and guard','Use your Abyss build','Bank your rewards']){await page.locator('#rift-walkthrough-next').click();await expect(page.locator('#rift-walkthrough-title')).toHaveText(title);}
 await page.locator('#rift-walkthrough-back').click();await expect(page.locator('#rift-walkthrough-title')).toHaveText('Use your Abyss build');await page.locator('#rift-walkthrough-next').click();await expect(page.locator('#rift-walkthrough-next')).toHaveText('Finish');await page.locator('#rift-walkthrough-next').click();
 expect((await(await page.request.get('/api/abyss/rift')).json()).run).toBeNull();await page.reload();await expect(page.locator('#rift-start')).toBeEnabled();await expect(page.locator('#rift-walkthrough')).toBeHidden();
});
