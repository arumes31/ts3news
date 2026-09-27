const {test,expect}=require('@playwright/test');

test('paused duration survives reload and is recorded separately on resume',async({page})=>{
 await page.goto('/abyss/rift');await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();await page.keyboard.press('Escape');await expect(page.locator('#rift-paused-badge')).toHaveText('Paused');
 const read=async()=>(await(await page.request.get('/api/abyss/rift')).json()).run;await expect.poll(async()=>(await read()).pause_started_ms||0).toBeGreaterThan(0);const paused=await read();
 await page.reload();await page.waitForTimeout(1100);const waiting=await read();expect(waiting.stats.seconds).toBe(paused.stats.seconds);expect(waiting.pause_started_ms).toBe(paused.pause_started_ms);
 await page.locator('#rift-start').click();await expect.poll(async()=>(await read()).stats.paused_seconds).toBeGreaterThanOrEqual(1);await page.keyboard.press('Escape');await expect(page.locator('#rift-paused-badge')).toHaveText('Paused');
 await page.locator('.rift-run-statistics > summary').click();await expect(page.locator('#rift-statistics dt').filter({hasText:'Paused seconds (completed pauses)'}).locator('xpath=following-sibling::dd[1]')).not.toHaveText('0');await expect(page.locator('.rift-run-statistics')).toContainText('an ongoing pause is added on resume');
});
