const {test,expect}=require('@playwright/test');
test.use({hasTouch:true,isMobile:true,viewport:{width:390,height:844}});

test('battery-saving preference persists without hiding critical combat cues',async({page})=>{
 await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();
 await page.locator('.rift-settings > summary').tap();
 await page.locator('#rift-display-preset').selectOption('lowPower');
 await expect(page.locator('#rift-preset-description')).toContainText('conserving battery');
 await expect(page.locator('#rift-render-rate')).toHaveValue('60');
 await page.locator('#rift-apply-preset').tap();
 const read=()=>page.evaluate(()=>{const d=window.RiftDisplay;return {fps:d.fps,particles:d.particles,motion:d.motionIntensity,flash:d.flashIntensity,health:d.healthBars,hazards:d.hazardLabels,patterns:d.hazardPatterns,shapes:d.projectileShapes};});
 const expected={fps:30,particles:false,motion:0,flash:0,health:true,hazards:true,patterns:true,shapes:true};
 expect(await read()).toEqual(expected);
 await page.reload();await expect(page.locator('#rift-start')).toBeEnabled();expect(await read()).toEqual(expected);
 await page.locator('.rift-settings > summary').tap();await expect(page.locator('#rift-render-rate')).toHaveValue('30');
 await page.locator('#rift-reset-display').tap();expect((await read()).fps).toBe(60);
});
