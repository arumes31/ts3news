const {test,expect}=require('@playwright/test');

test('basic combo peak records confirmed attacks and survives reload',async({page})=>{
 await page.goto('/abyss/rift');await page.locator('#rift-start').click();
 const peak=page.locator('#rift-statistics dt').filter({hasText:'Highest basic combo strike (of 3)'}).locator('xpath=following-sibling::dd[1]');
 await page.keyboard.down('j');await expect(peak).toHaveText('3');await page.keyboard.up('j');
 await page.keyboard.press('Escape');await expect(page.locator('#rift-paused-badge')).toHaveText('Paused');
 const run=(await(await page.request.get('/api/abyss/rift')).json()).run;expect(run.stats.highest_combo).toBe(3);expect(run.stats.attacks).toBeGreaterThanOrEqual(3);
 await page.reload();await page.locator('.rift-run-statistics > summary').click();await expect(peak).toHaveText('3');await expect(page.locator('.rift-run-statistics')).toContainText('including attacks that miss');
 await page.setViewportSize({width:390,height:844});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
});
