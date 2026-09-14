const {test,expect}=require('@playwright/test');

test('personal HUD records can be hidden without losing saved records or combat breakdowns',async({page})=>{
 await page.goto('/abyss/rift?scenario=checkpoint&room=final');await page.locator('#rift-auto').uncheck();await page.locator('#rift-start').click();await page.locator('#rift-next').click();
 await expect(page.locator('#rift-clear-result')).toBeVisible();const before=(await(await page.request.get('/api/abyss/rift')).json()).run;
 await page.locator('.rift-settings > summary').click();await page.locator('#rift-personal-records').uncheck();await expect(page.locator('#rift-clear-result')).toBeHidden();
 await page.locator('.rift-run-statistics > summary').click();await expect(page.locator('#rift-career-statistics')).toBeHidden();await expect(page.locator('#rift-statistics dt').filter({hasText:'Best mission clear streak'})).toBeHidden();await expect(page.locator('#rift-statistics dt').filter({hasText:'Damage dealt'})).toBeVisible();await expect(page.locator('#rift-history-1')).toContainText('Most HP at finish');
 await page.evaluate(()=>history.replaceState(null,'','/abyss/rift'));await page.reload();await page.locator('.rift-settings > summary').click();await expect(page.locator('#rift-personal-records')).not.toBeChecked();await expect(page.locator('#rift-clear-result')).toBeHidden();
 expect((await(await page.request.get('/api/abyss/rift')).json()).run).toEqual(before);
 await page.locator('#rift-reset-display').click();await expect(page.locator('#rift-personal-records')).toBeChecked();await expect(page.locator('#rift-clear-result')).toBeVisible();
 await page.locator('.rift-run-statistics > summary').click();await expect(page.locator('#rift-career-statistics')).toBeVisible();
});
