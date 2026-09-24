const {test,expect}=require('@playwright/test');
test('truncated banking JSON recovers committed rewards without a duplicate write',async({page})=>{
 const errors=[];page.on('pageerror',error=>errors.push(error.message));
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('#rift-auto').uncheck();
 const writes=[];let damaged=false;
 await page.route('**/api/abyss/rift',async route=>{
  if(route.request().method()==='POST')writes.push(route.request().postDataJSON().kind);
  if(!damaged&&route.request().method()==='POST'&&route.request().postDataJSON().kind==='exit'){damaged=true;const response=await route.fetch(),body=await response.text();await route.fulfill({status:200,contentType:'application/json',body:body.slice(0,Math.floor(body.length/2))});}else await route.continue();
 });
 await page.locator('#rift-start').click();await page.locator('#rift-exit').click();await expect(page.locator('#rift-start')).toHaveText('Recover expedition');await expect(page.locator('#rift-banking-status')).toContainText('unconfirmed');await expect(page.locator('#rift-banked')).toHaveText('0 gold · 0 items');
 const committed=(await(await page.request.get('/api/abyss/rift')).json()).run;expect(committed.banked_gold).toBe(30);expect(committed.banked_items).toHaveLength(1);const before=[...writes];
 await page.locator('#rift-start').click();await expect(page.locator('#rift-banked')).toHaveText('30 gold · 1 item');expect(writes).toEqual(before);expect(writes.filter(kind=>kind==='exit')).toHaveLength(1);expect((await(await page.request.get('/api/abyss/rift')).json()).run).toEqual(committed);expect(errors).toEqual([]);
});

test('truncated initial JSON offers read-only recovery',async({page})=>{
 let damaged=false;const writes=[];
 await page.route('**/api/abyss/rift',async route=>{
  if(route.request().method()==='POST')writes.push(route.request().postDataJSON().kind);
  if(!damaged&&route.request().method()==='GET'){damaged=true;await route.fulfill({status:200,contentType:'application/json',body:'{"ok":true,"run":'});}else await route.continue();
 });
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toHaveText('Retry loading');const saved=(await(await page.request.get('/api/abyss/rift')).json()).run;
 await page.locator('#rift-start').click();await expect(page.locator('#rift-start')).toBeEnabled();await expect(page.locator('#rift-start')).not.toHaveText('Retry loading');expect(writes).toEqual([]);expect((await(await page.request.get('/api/abyss/rift')).json()).run).toEqual(saved);
});
