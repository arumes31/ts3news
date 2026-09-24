const {test,expect}=require('@playwright/test');
test('charges stay visible during countdown and a pending tier advance',async({page})=>{
 await page.goto('/abyss/rift?scenario=checkpoint&charges=2');await expect(page.locator('#rift-start')).toBeEnabled();
 let release,observed;const held=new Promise(resolve=>release=resolve),pending=new Promise(resolve=>observed=resolve);
 await page.route('**/api/abyss/rift',async route=>{if(route.request().method()==='POST'&&route.request().postDataJSON().kind==='advance'){observed();await held;}await route.continue();});
 await page.locator('#rift-start').click();await expect(page.locator('#rift-transition')).toContainText('Resolve 2/3');await expect(page.locator('#rift-transition')).toContainText('Next:');await pending;
 await expect(page.locator('#rift-transition')).toContainText('Banking rewards');await expect(page.locator('#rift-transition')).toContainText('Resolve 2/3');await expect(page.locator('#rift-resource')).toContainText('Resolve 2/3');
 await page.setViewportSize({width:390,height:1200});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);await page.locator('#rift-transition').screenshot({path:'test-results/transition-charges-mobile.png'});
 release();const read=async()=>(await(await page.request.get('/api/abyss/rift')).json()).run;await expect.poll(async()=>(await read()).room).toBe(1);expect((await read()).resource).toBe(2);await expect(page.locator('#rift-resource')).toContainText('Resolve 2/3');await expect(page.locator('#rift-transition')).toBeHidden();
});

test('boss-room confirmation keeps charges readable without covering controls',async({page})=>{
 await page.addInitScript(()=>localStorage.setItem('riftPauseBossRoom','true'));await page.goto('/abyss/rift?scenario=checkpoint&room=preboss&charges=3');await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('#rift-start').click();await expect(page.locator('#rift-transition')).toContainText('Boss room ahead');await expect(page.locator('#rift-transition')).toContainText('Resolve 3/3');await page.setViewportSize({width:390,height:1200});
 const banner=await page.locator('#rift-transition').boundingBox(),button=await page.locator('#rift-next').boundingBox();expect(banner).toBeTruthy();expect(button).toBeTruthy();expect(button.y+button.height).toBeLessThanOrEqual(banner.y);expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
});
