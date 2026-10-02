const {test,expect}=require('@playwright/test');
test('stale second-tab exit cannot overwrite a checkpoint advanced by the first tab',async({page,context})=>{
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('#rift-auto').uncheck();
 const second=await context.newPage();
 try{
  await second.goto('/abyss/rift');await expect(second.locator('#rift-start')).toBeEnabled();
  const read=async tab=>tab.evaluate(async()=> (await(await fetch('/api/abyss/rift')).json()).run);
  const before=await read(page),other=await read(second);expect(other.id).toBe(before.id);expect(other.revision).toBe(before.revision);expect(before.status).toBe('cleared');
  const send=async(tab,kind,id)=>tab.evaluate(async({kind,id,run})=>{
   const response=await fetch('/api/abyss/rift',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({kind,run_id:run.id,revision:run.revision+1,request_id:id})});
   return {status:response.status,data:await response.json()};
  },{kind,id,run:before});
  const advanced=await send(page,'advance','first-tab-advance-request');expect(advanced.status).toBe(200);expect(advanced.data.run.room).toBe(before.room+1);expect(advanced.data.run.status).toBe('fighting');
  const retry=await send(page,'advance','first-tab-advance-request');expect(retry.status).toBe(200);expect(retry.data.run).toEqual(advanced.data.run);
  const conflict=await send(second,'exit','second-tab-stale-exit');expect(conflict.status).toBe(409);expect(conflict.data.ok).toBe(false);
  expect(await read(second)).toEqual(advanced.data.run);
  let posts=0;await second.route('**/api/abyss/rift',route=>{if(route.request().method()==='POST')posts++;return route.continue();});
  await second.locator('#rift-start').click();
  await expect(second.locator('#rift-overlay-kicker')).toHaveText('SAVE CONFLICT');
  await expect(second.locator('#rift-overlay-copy')).toContainText('Another tab may have advanced it');
  const stopped=posts;await second.waitForTimeout(300);expect(posts).toBe(stopped);
  await second.getByRole('button',{name:'Reload saved expedition',exact:true}).click();
  await expect(second.locator('#rift-start')).toHaveText('Resume expedition');
  await second.waitForTimeout(300);expect(posts).toBe(stopped);
  await expect(second.locator('#rift-banked')).toHaveText('30 gold · 1 item');
  const recovered=await read(second);expect(recovered.room).toBe(advanced.data.run.room);expect(recovered.status).toBe('fighting');expect(recovered.revision).toBe(advanced.data.run.revision);expect(recovered.banked_items).toEqual(advanced.data.run.banked_items);
  await second.locator('#rift-start').click();await expect(second.locator('#rift-overlay')).toBeHidden();
  await second.keyboard.press('Escape');await expect.poll(async()=>(await read(second)).paused).toBe(true);
  const resumed=await read(second);expect(resumed.revision).toBeGreaterThan(recovered.revision);expect(resumed.banked_gold).toBe(recovered.banked_gold);expect(resumed.banked_items).toEqual(recovered.banked_items);
 }finally{await second.close();}
});
