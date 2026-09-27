const {test,expect}=require('@playwright/test');
test('expired initial read offers sign-in and read-only recovery',async({page})=>{
 let expired=true,posts=0;
 await page.route('**/api/abyss/rift',route=>{if(route.request().method()==='POST')posts++;if(expired)return route.fulfill({status:401,contentType:'application/json',body:'{"ok":false,"error":"unauthenticated"}'});return route.continue();});
 await page.goto('/abyss/rift');await expect(page.locator('#rift-overlay-kicker')).toHaveText('SESSION EXPIRED');await expect(page.locator('#rift-overlay-copy')).toContainText('Sign in again');
 await expect(page.locator('#rift-sign-in')).toBeVisible();await expect(page.locator('#rift-sign-in')).toHaveAttribute('href','/login?next=%2Fabyss%2Frift');
 await page.setViewportSize({width:390,height:844});await page.locator('#rift-sign-in').scrollIntoViewIfNeeded();const bounds=await page.locator('#rift-sign-in').boundingBox();expect(bounds.height).toBeGreaterThanOrEqual(44);expect(bounds.width).toBeGreaterThanOrEqual(44);expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
 expired=false;await page.getByRole('button',{name:'Check session again',exact:true}).click();await expect(page.locator('#rift-start')).toBeEnabled();await expect(page.locator('#rift-sign-in')).toBeHidden();expect(posts).toBe(0);
});
for(const code of [401,500])test('combat HTTP '+code+' explains the failure and requires explicit resume',async({page})=>{
 await page.goto('/abyss/rift?scenario=spawn-hazards');await expect(page.locator('#rift-start')).toBeEnabled();let rejected=false,steps=0,posts=0;
 await page.route('**/api/abyss/rift',route=>{if(route.request().method()==='POST'){posts++;if(route.request().postDataJSON().kind==='step'){steps++;if(!rejected){rejected=true;return route.fulfill({status:code,contentType:'application/json',body:'{"ok":false}'});}}}return route.continue();});
 await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay-kicker')).toHaveText(code===401?'SESSION EXPIRED':'CONNECTION PAUSED');
 if(code===401){await expect(page.locator('#rift-sign-in')).toBeVisible();await expect(page.locator('#rift-overlay-copy')).toContainText('Sign in again');}else await expect(page.locator('#rift-sign-in')).toBeHidden();
 const stopped=posts;await page.waitForTimeout(400);expect(posts).toBe(stopped);
 await page.locator('#rift-start').click();await expect(page.locator('#rift-start')).toHaveText('Resume expedition');await expect(page.locator('#rift-sign-in')).toBeHidden();await page.waitForTimeout(400);expect(posts).toBe(stopped);
 await page.locator('#rift-start').click();await expect.poll(()=>steps).toBeGreaterThan(1);await page.keyboard.press('Escape');
});

test('session expiry during banking keeps delivery explicitly unconfirmed',async({page})=>{
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('#rift-auto').uncheck();
 await page.locator('#rift-start').click();await expect(page.locator('#rift-next')).toBeEnabled();
 await page.route('**/api/abyss/rift',route=>route.request().method()==='POST'&&route.request().postDataJSON().kind==='next'?route.fulfill({status:401,contentType:'application/json',body:'{"ok":false}'}):route.continue());
 await page.locator('#rift-next').click();await expect(page.locator('#rift-overlay-kicker')).toHaveText('SESSION EXPIRED');await expect(page.locator('#rift-overlay-copy')).toContainText('Reward delivery is unconfirmed');await expect(page.locator('#rift-sign-in')).toBeVisible();
});
