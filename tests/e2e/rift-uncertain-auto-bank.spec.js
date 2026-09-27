const {test,expect}=require('@playwright/test');
for(const committed of [false,true])test('uncertain automatic banking stays stopped after '+(committed?'committed':'uncommitted')+' recovery',async({page})=>{
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();await expect(page.locator('#rift-auto')).toBeChecked();const saved=async()=>(await(await page.request.get('/api/abyss/rift')).json()).run;const before=await saved();let failed=false;const writes=[];
 await page.route('**/api/abyss/rift',async route=>{
  if(route.request().method()==='POST'){const body=route.request().postDataJSON();writes.push(body.kind);if(body.kind==='advance'&&!failed){failed=true;if(committed)await route.fetch();await route.abort();return;}}
  await route.continue();
 });
 await page.locator('#rift-start').click();await expect(page.locator('#rift-start')).toHaveText('Recover expedition');await expect(page.locator('#rift-banking-status')).toContainText('delivery is unconfirmed');const stopped=writes.length;
 await page.waitForTimeout(1600);expect(writes).toHaveLength(stopped);await page.locator('#rift-start').click();await expect(page.locator('#rift-start')).toHaveText('Resume expedition');await expect(page.locator('#rift-auto')).toBeChecked();await expect(page.locator('#rift-transition')).toBeHidden();
 await page.waitForTimeout(1600);expect(writes).toHaveLength(stopped);const recovered=await saved();expect(recovered.room).toBe(before.room+(committed?1:0));expect(recovered.banked_gold).toBe(committed?30:0);expect(recovered.banked_items).toHaveLength(committed?1:0);if(committed)expect(await page.evaluate(()=>sessionStorage.getItem('riftPendingBank:campaign'))).toBeNull();
 await page.locator('#rift-start').click();await expect.poll(()=>writes.length).toBeGreaterThan(stopped);
 if(!committed)await expect.poll(async()=>(await saved()).room).toBe(before.room+1);else await expect.poll(()=>writes.includes('step')).toBe(true);
 await page.keyboard.press('Escape');expect(writes.filter(kind=>kind==='advance')).toHaveLength(committed?1:2);const resumed=await saved();expect(resumed.banked_gold).toBe(30);expect(resumed.banked_items).toHaveLength(1);
});
