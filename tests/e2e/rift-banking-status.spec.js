const {test,expect}=require('@playwright/test');

for(const committed of [false,true])test('uncertain bank recovery reloads '+(committed?'committed':'uncommitted')+' rewards without another write',async({page})=>{
 await page.goto('/abyss/rift?scenario=checkpoint');await page.locator('#rift-auto').uncheck();
 let release;const held=new Promise(resolve=>release=resolve);let calls=0;
 await page.route('**/api/abyss/rift',async route=>{
  if(route.request().method()==='POST'&&route.request().postDataJSON().kind==='exit'){calls++;await held;if(committed)await route.fetch();await route.abort('failed');}else await route.continue();
 });
 await page.locator('#rift-start').click();await page.locator('#rift-exit').click();
 try{await expect(page.locator('#rift-banking-status')).toContainText('not confirmed yet');}finally{release();}
 await expect(page.locator('#rift-start')).toHaveText('Recover expedition');await expect(page.locator('#rift-banking-status')).toContainText('delivery is unconfirmed');
 await page.locator('#rift-start').click();await expect(page.locator('#rift-banking-status')).toContainText('Saved reward state loaded');
 await expect(page.locator('#rift-banked')).toHaveText(committed?'30 gold · 1 item':'0 gold · 0 items');expect(calls).toBe(1);
 const career=label=>page.locator('#rift-career-statistics dt').filter({hasText:label}).locator('xpath=following-sibling::dd[1]');await expect(career('Gold banked')).toHaveText(committed?'30':'0');await expect(career('Gear pieces banked')).toHaveText(committed?'1':'0');
 const run=(await(await page.request.get('/api/abyss/rift')).json()).run;expect(run.banked_gold).toBe(committed?30:0);
 await page.setViewportSize({width:390,height:844});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
});
