const {test,expect}=require('@playwright/test');
test('recovery shows the server room after a committed continuation loses its response',async({page})=>{
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('#rift-auto').uncheck();
 const before=(await(await page.request.get('/api/abyss/rift')).json()).run;
 await page.locator('#rift-start').click();await expect(page.locator('#rift-room-actions')).toBeVisible();
 let committed=null,nextRequests=0;
 await page.route('**/api/abyss/rift',async route=>{
  if(route.request().postDataJSON()?.kind==='next'){
   nextRequests++;const response=await route.fetch();expect(response.ok()).toBe(true);committed=(await response.json()).run;
   await route.abort();return;
  }
  await route.continue();
 });
 await page.locator('#rift-next').click();await expect(page.locator('#rift-start')).toHaveText('Recover expedition');
 expect(committed.room).toBe(before.room+1);await expect(page.locator('#rift-room')).toContainText('Tier '+(before.room+1)+'/3');
 await page.locator('#rift-start').click();await expect(page.locator('#rift-start')).toHaveText('Resume expedition');
 await expect(page.locator('#rift-room')).toContainText('Tier '+(committed.room+1)+'/3');
 const route=page.locator('.rift-route > li');await expect(route.nth(committed.room)).toHaveClass('current');
 await expect(route.nth(before.room)).toHaveClass('done');await expect(page.locator('#rift-room-actions')).toBeHidden();
 expect(nextRequests).toBe(1);
 const recovered=(await(await page.request.get('/api/abyss/rift')).json()).run;
 expect(recovered.id).toBe(committed.id);expect(recovered.room).toBe(committed.room);expect(recovered.banked_items).toEqual(committed.banked_items);
 await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();
 await expect(page.locator('#rift-room')).toContainText('Tier '+(committed.room+1)+'/3');await page.locator('#rift-pause').click();
 expect(nextRequests).toBe(1);
});
