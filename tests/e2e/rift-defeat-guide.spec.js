const {test,expect}=require('@playwright/test');

test('defeat guide opens fight review and mission choice without restarting and remembers dismissal',async({page})=>{
 let writes=0;page.on('request',request=>{if(request.url().includes('/api/abyss/rift')&&request.method()==='POST')writes++;});
 await page.route('**/api/abyss/rift',async route=>{const response=await route.fetch(),data=await response.json();if(data.run){data.run.status='defeated';data.run.player.hp=0;data.run.gold=0;data.run.drops=[];data.hazard_hit_damage=0;}await route.fulfill({response,json:data});});
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-defeat-guide')).toBeVisible();await expect(page.locator('#rift-defeat-guide')).toContainText('equipped gear remain safe');await expect(page.locator('#rift-defeat-guide')).toContainText('Choose skills');
 await page.locator('#rift-defeat-review').click();await expect(page.locator('.rift-run-statistics')).toHaveAttribute('open','');await page.locator('#rift-defeat-missions').click();await expect(page.locator('#rift-region-overview')).toBeVisible();expect(writes).toBe(0);
 await page.setViewportSize({width:390,height:844});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);await page.locator('#rift-defeat-dismiss').click();await expect(page.locator('#rift-defeat-guide')).toBeHidden();await page.reload();await expect(page.locator('#rift-start')).toBeEnabled();await expect(page.locator('#rift-defeat-guide')).toBeHidden();
});
