const {test,expect}=require('@playwright/test');

test('clear streak appears in statistics, survives reload and carries into replay',async({page})=>{
 await page.goto('/abyss/rift?scenario=checkpoint&room=final');await page.locator('#rift-auto').uncheck();await page.locator('#rift-start').click();await page.locator('#rift-next').click();
 const value=label=>page.locator('#rift-statistics dt').filter({hasText:label}).locator('xpath=following-sibling::dd[1]');
 await expect(value('Current mission clear streak')).toHaveText('1');await expect(value('Best mission clear streak')).toHaveText('1');
 await page.evaluate(()=>history.replaceState(null,'','/abyss/rift'));await page.reload();await expect(value('Current mission clear streak')).toHaveText('1');
 await page.locator('#rift-replay').click();await expect(page.locator('#rift-overlay')).toBeHidden();await page.keyboard.press('Escape');await expect(page.locator('#rift-paused-badge')).toHaveText('Paused');
 const run=(await(await page.request.get('/api/abyss/rift')).json()).run;expect(run.clear_streak).toBe(1);expect(run.best_clear_streak).toBe(1);
 await page.locator('.rift-run-statistics > summary').click();await expect(value('Best mission clear streak')).toHaveText('1');await expect(page.locator('.rift-run-statistics')).toContainText('early exit');
 await page.setViewportSize({width:390,height:844});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
});
