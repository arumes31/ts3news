const {test,expect}=require('@playwright/test');

test('paused badge persists while browsing and after reload, then clears on resume',async({page})=>{
  await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();await expect(page.locator('#rift-paused-badge')).toBeHidden();
  await page.locator('#rift-start').click();await expect(page.locator('#rift-paused-badge')).toBeHidden();await page.keyboard.press('Escape');
  await expect(page.locator('#rift-paused-badge')).toHaveText('Paused');await expect(page.locator('#rift-paused-badge')).toBeVisible();
  await page.locator('#rift-loadout-preview').click();await expect(page.locator('#rift-glossary-search')).toBeFocused();await expect(page.locator('#rift-paused-badge')).toBeVisible();
  await page.reload();await expect(page.locator('#rift-paused-badge')).toHaveText('Paused');
  await page.locator('#rift-start').click();await expect(page.locator('#rift-paused-badge')).toBeHidden();await page.keyboard.press('Escape');
});

test('failed pause displays a local stop without claiming server confirmation',async({page})=>{
  await page.goto('/abyss/rift');await page.locator('#rift-start').click();
  await page.route('**/api/abyss/rift',async route=>{if(route.request().method()==='POST'&&route.request().postDataJSON().kind==='pause')await route.abort();else await route.continue();});
  await page.keyboard.press('Escape');await expect(page.locator('#rift-start')).toHaveText('Recover expedition');
  await expect(page.locator('#rift-paused-badge')).toHaveText('Not running');await expect(page.locator('#rift-paused-badge')).toBeVisible();
  await page.setViewportSize({width:390,height:844});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
});
