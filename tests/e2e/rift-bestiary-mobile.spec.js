const {test,expect}=require('@playwright/test');

test('mobile inspection starts with a concise summary and expandable complete stats',async({page})=>{
 await page.setViewportSize({width:390,height:844});await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('.rift-bestiary > summary').click();await page.locator('#rift-monsters button').first().click();await expect(page.locator('#rift-monster-quick-summary')).toBeVisible();await expect(page.locator('#rift-monster-quick-summary')).toContainText('Windup');await expect(page.locator('#rift-monster-stat-details')).not.toHaveAttribute('open','');await expect(page.locator('#rift-monster-stats')).toBeHidden();await page.locator('#rift-monster-stat-details > summary').focus();await page.keyboard.press('Enter');await expect(page.locator('#rift-monster-stats')).toBeVisible();await expect(page.locator('#rift-monster-stats')).toContainText('Health');expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);expect((await(await page.request.get('/api/abyss/rift')).json()).run).toBeNull();
});

test('desktop monster inspection keeps full stats expanded',async({page})=>{
 await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('.rift-bestiary > summary').click();await page.locator('#rift-monsters button').first().click();await expect(page.locator('#rift-monster-stat-details')).toHaveAttribute('open','');await expect(page.locator('#rift-monster-stats')).toBeVisible();
});
