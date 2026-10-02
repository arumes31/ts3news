const {test,expect}=require('@playwright/test');
test.use({hasTouch:true,isMobile:true,viewport:{width:390,height:844}});

test('mission search keeps typing focus and filtering in a shortened mobile viewport',async({page})=>{
 await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();
 const search=page.locator('#rift-mission-search');await search.tap();
 await expect(search).toHaveAttribute('inputmode','search');await expect(search).toHaveAttribute('enterkeyhint','search');
 await expect(search).toHaveAttribute('autocapitalize','off');await expect(search).toHaveAttribute('spellcheck','false');
 await search.pressSequentially('97');await expect(search).toHaveValue('97');await expect(search).toBeFocused();
 await expect(page.locator('#rift-levels [data-level="97"]')).toBeVisible();
 const filtered=await page.locator('#rift-levels [data-level]:visible').count();expect(filtered).toBeLessThan(100);expect(filtered).toBeGreaterThan(0);
 // Reduced layout viewport is a repeatable keyboard-space proxy, not a native IME.
 await page.setViewportSize({width:390,height:480});await search.scrollIntoViewIfNeeded();await expect(search).toBeFocused();
 await search.press('Enter');await expect(search).toHaveValue('97');await expect(search).toBeFocused();
 const box=await search.boundingBox();expect(box.height).toBeGreaterThanOrEqual(48);
 expect(await search.evaluate(node=>getComputedStyle(node).fontSize)).toBe('16px');
 await search.fill('');await expect(page.locator('#rift-levels [data-level]:visible')).toHaveCount(100);
 expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
});
