const {test,expect}=require('@playwright/test');

test('compact settings summary follows saved choices, resets and system motion',async({page})=>{
 await page.emulateMedia({reducedMotion:'no-preference'});await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();
 const summary=page.locator('#rift-settings-summary');await expect(summary).toContainText('Full HUD');await expect(summary).toContainText('60 FPS');
 await page.locator('.rift-settings > summary').click();await page.locator('#rift-compact-hud').check();await page.locator('#rift-text-scale').selectOption('1.25');await page.locator('#rift-shake-intensity').selectOption('0.5');await page.locator('#rift-render-rate').selectOption('30');
 await page.locator('#rift-interface-volume').evaluate(el=>el.value='0');await page.locator('#rift-interface-volume').dispatchEvent('input');await page.locator('#rift-mono-audio').check();
 await expect(summary).toContainText('Compact HUD · Text 125% · 30 FPS');await expect(summary).toContainText('Shake Gentle');await expect(summary).toContainText('Interface 0%');await expect(summary).toContainText('Mono');
 await page.locator('.rift-settings > summary').click();await expect(summary).toBeVisible();await page.reload();await expect(summary).toContainText('Compact HUD · Text 125% · 30 FPS');await expect(summary).toContainText('Interface 0%');
 await page.emulateMedia({reducedMotion:'reduce'});await expect(summary).toContainText('Reduced effects · Shake Off');await page.locator('#rift-sound').click();await expect(summary).toContainText('Sound muted');
 await page.locator('.rift-settings > summary').click();await page.locator('#rift-reset-display').click();await page.locator('#rift-reset-audio').click();await expect(summary).toContainText('Full HUD · Text 100% · 60 FPS');await expect(summary).toContainText('Interface 65%');await expect(summary).toContainText('Sound muted');
 await page.locator('.rift-settings > summary').click();await page.setViewportSize({width:390,height:844});await expect(summary).toBeVisible();expect(await summary.evaluate(el=>el.scrollWidth<=el.clientWidth)).toBe(true);expect((await(await page.request.get('/api/abyss/rift')).json()).run).toBeNull();
});
