const {test,expect}=require('@playwright/test');

for(const committed of [false,true])test('uncertain bank recovery reloads '+(committed?'committed':'uncommitted')+' rewards without another write',async({page})=>{
 await page.goto('/abyss/rift?scenario=checkpoint');await page.locator('#rift-auto').uncheck();
 let release;const held=new Promise(resolve=>release=resolve);let calls=0;const writes=[];
 await page.route('**/api/abyss/rift',async route=>{
  if(route.request().method()==='POST')writes.push(route.request().postDataJSON().kind);
  if(route.request().method()==='POST'&&route.request().postDataJSON().kind==='exit'){calls++;await held;if(committed)await route.fetch();await route.abort('failed');}else await route.continue();
 });
 await page.locator('#rift-start').click();await page.locator('#rift-exit').click();
 try{await expect(page.locator('#rift-banking-status')).toContainText('not confirmed yet');await expect(page.locator('#rift-banked')).toHaveText('0 gold · 0 items');await expect(page.locator('#rift-gold')).toHaveText('30');await expect(page.locator('#rift-loot-count')).toHaveText('1 item pending');}finally{release();}
 await expect(page.locator('#rift-start')).toHaveText('Recover expedition');await expect(page.locator('#rift-banking-status')).toContainText('delivery is unconfirmed');
 const writesBeforeRecovery=writes.length;
 await page.locator('#rift-start').click();await expect(page.locator('#rift-banking-status')).toContainText('Saved reward state loaded');
 expect(writes).toHaveLength(writesBeforeRecovery);
 await expect(page.locator('#rift-banked')).toHaveText(committed?'30 gold · 1 item':'0 gold · 0 items');expect(calls).toBe(1);
 const career=label=>page.locator('#rift-career-statistics dt').filter({hasText:label}).locator('xpath=following-sibling::dd[1]');await expect(career('Gold banked')).toHaveText(committed?'30':'0');await expect(career('Gear pieces banked')).toHaveText(committed?'1':'0');
 const run=(await(await page.request.get('/api/abyss/rift')).json()).run;expect(run.banked_gold).toBe(committed?30:0);
 await page.evaluate(()=>history.replaceState(null,'','/abyss/rift'));
 for(let reload=0;reload<2;reload++){
  await page.reload();await expect(page.locator('#rift-start')).toBeEnabled();
  await expect(page.locator('#rift-banked')).toHaveText(committed?'30 gold · 1 item':'0 gold · 0 items');
  const saved=(await(await page.request.get('/api/abyss/rift')).json()).run;
  for(const key of ['id','revision','status','banked_gold','banked_items','banked_loot','banked_at_ms','drops'])expect(saved[key]).toEqual(run[key]);
  expect(writes).toHaveLength(writesBeforeRecovery);expect(calls).toBe(1);
 }
 await page.setViewportSize({width:390,height:844});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
});
