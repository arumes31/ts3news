const {test,expect}=require('@playwright/test');
test('creature family filters shared Abyss artwork groups and resets with other filters',async({page})=>{
 await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('details').filter({has:page.locator('#rift-monsters')}).locator(':scope > summary').click();
 const family=page.locator('#rift-monster-family'),cards=page.locator('#rift-monsters > article:visible');await expect(family).toBeVisible();const total=await cards.count();expect(total).toBeGreaterThan(10);
 await family.selectOption('Rat');expect(await cards.count()).toBeGreaterThan(1);expect(await cards.evaluateAll(nodes=>nodes.every(n=>n.dataset.family==='Rat'))).toBe(true);await expect(page.locator('#rift-monster-matches')).toContainText((await cards.count())+' of '+total);
 await cards.first().getByRole('button',{name:/Inspect/}).click();await expect(page.locator('#rift-monster-detail')).toContainText('Creature family');await expect(page.locator('#rift-monster-detail')).toContainText('Rat');
 await page.locator('#rift-monster-search').fill('not-a-real-creature');await expect(cards).toHaveCount(0);await expect(page.locator('#rift-monsters-empty')).toBeVisible();await page.locator('#rift-monster-clear').click();await expect(family).toHaveValue('');await expect(cards).toHaveCount(total);
 await page.setViewportSize({width:390,height:1600});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
});
