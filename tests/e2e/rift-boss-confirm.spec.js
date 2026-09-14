const {test,expect}=require('@playwright/test');

test('boss confirmation persists and holds the final tier until explicit continuation',async({page})=>{
 await page.goto('/abyss/rift?scenario=checkpoint&room=final');await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('.rift-settings > summary').click();await page.locator('#rift-confirm-boss').check();await page.evaluate(()=>history.replaceState(null,'','/abyss/rift'));await page.reload();await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('.rift-settings > summary').click();await expect(page.locator('#rift-confirm-boss')).toBeChecked();await page.locator('.rift-settings > summary').click();const before=(await(await page.request.get('/api/abyss/rift')).json()).run;
 await page.locator('#rift-start').click();await expect(page.locator('#rift-transition')).toContainText('Confirm to bank');await page.waitForTimeout(1600);const held=(await(await page.request.get('/api/abyss/rift')).json()).run;expect(held.level.id).toBe(before.level.id);expect(held.status).toBe('cleared');await page.locator('#rift-next').click();await expect.poll(async()=>(await(await page.request.get('/api/abyss/rift')).json()).run.level.id).toBe(before.level.id+1);
});

test('boss confirmation does not stop automatic progression in ordinary tiers',async({page})=>{
 await page.addInitScript(()=>localStorage.setItem('riftConfirmBoss','true'));await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();const before=(await(await page.request.get('/api/abyss/rift')).json()).run;await page.locator('#rift-start').click();await expect.poll(async()=>(await(await page.request.get('/api/abyss/rift')).json()).run.room).toBe(before.room+1);
});
