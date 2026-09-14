const {test,expect}=require('@playwright/test');

test('campaign displays recorded flawless tiers and omits unknown older records',async({page})=>{
 let recorded=true;
 await page.route('**/api/abyss/rift',async route=>{const response=await route.fetch(),data=await response.json();if(data.run&&recorded)data.run.mission_history['1'].flawless_tiers=[1,3];await route.fulfill({response,json:data});});
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-history-1')).toContainText('Flawless tiers 1, 3');
 await page.setViewportSize({width:390,height:844});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
 recorded=false;await page.reload();await expect(page.locator('#rift-start')).toBeEnabled();await expect(page.locator('#rift-history-1')).not.toContainText('Flawless tiers');
});
