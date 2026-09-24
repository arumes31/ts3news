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
  const conflict=await send(second,'exit','second-tab-stale-exit');expect(conflict.status).toBe(200);expect(conflict.data.run).toEqual(advanced.data.run);
  expect(await read(second)).toEqual(advanced.data.run);
  await second.reload();await expect(second.locator('#rift-start')).toBeEnabled();
  await expect(second.locator('#rift-banked')).toHaveText('30 gold · 1 item');
  const recovered=await read(second);expect(recovered.room).toBe(advanced.data.run.room);expect(recovered.status).toBe('fighting');expect(recovered.revision).toBe(advanced.data.run.revision);expect(recovered.banked_items).toEqual(advanced.data.run.banked_items);
 }finally{await second.close();}
});
