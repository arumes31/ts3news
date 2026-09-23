const {test,expect}=require('@playwright/test');
test('preview optional objectives and retain failed ability goal while paused',async({page})=>{
 await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();
 const panel=page.locator('#rift-optional-objectives');await expect(panel).toContainText('Swift clear');await expect(panel).toContainText('180 combat seconds');
 await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();await page.keyboard.press('Digit1');
 const saved=async()=>(await(await page.request.get('/api/abyss/rift')).json()).run;
 await expect.poll(async()=>(await saved()).objectives.entries.find(e=>e.id==='basic_only').status).toBe('failed');
 await page.keyboard.press('Escape');await expect(page.locator('#rift-paused-badge')).toHaveText('Paused');
 await expect(panel.locator('[data-objective-id="basic_only"]')).toContainText('Used an ability.');await expect(panel).toContainText('Paused');expect((await saved()).status).toBe('fighting');
 await page.reload();await expect(page.locator('#rift-start')).toBeEnabled();await expect(panel.locator('[data-objective-id="basic_only"]')).toContainText('Failed');
});

test('completed objective results survive advancement and reload',async({page})=>{
 await page.goto('/abyss/rift?scenario=objective-results');await expect(page.locator('#rift-start')).toBeEnabled();
 await expect(page.locator('#rift-objectives-list [data-state="complete"]')).toHaveCount(4);
 await page.locator('#rift-start').click();
 await expect.poll(async()=>(await(await page.request.get('/api/abyss/rift')).json()).run.level.id).toBe(2);
 await page.keyboard.press('Escape');await page.evaluate(()=>history.replaceState(null,'','/abyss/rift'));await page.reload();await expect(page.locator('#rift-start')).toBeEnabled();
 await expect(page.locator('#rift-objectives-last-title')).toContainText('Previous mission 1');await page.locator('#rift-objectives-last-title').click();await expect(page.locator('#rift-objectives-last-list [data-state="complete"]')).toHaveCount(4);
 await page.setViewportSize({width:390,height:844});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
 await page.locator('#rift-optional-objectives').screenshot({path:'test-results/objectives-mobile.png'});
 await page.goto('/abyss/rift?practice=boss');await expect(page.locator('#rift-start')).toBeEnabled();await expect(page.locator('#rift-optional-objectives')).toBeHidden();
});
