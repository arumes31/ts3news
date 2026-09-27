const {test,expect}=require('@playwright/test');
for(const width of [1280,390])test('shortened receipt retains full totals and explains history at '+width+'px',async({page})=>{
 await page.setViewportSize({width,height:900});
 await page.route('**/api/abyss/rift',async route=>{
  const response=await route.fetch(),data=await response.json();
  if(data.run){data.run.banked_items_total=1234;data.run.banked_items=data.run.banked_items.slice(-2);data.run.banked_loot=data.run.banked_loot.slice(-2);}
  await route.fulfill({response,json:data});
 });
 await page.goto('/abyss/rift?scenario=long-receipt');await expect(page.locator('#rift-start')).toBeEnabled();
 await expect(page.locator('#rift-banked')).toContainText('1,234 items');
 await page.locator('#rift-receipt > summary').click();
 await expect(page.locator('#rift-receipt-total')).toContainText('1,234 items safely banked');
 await expect(page.locator('#rift-receipt-limit')).toContainText('Showing the most recent 2 of 1,234 banked items');
 await page.evaluate(()=>Object.defineProperty(navigator,'clipboard',{configurable:true,value:{writeText:async text=>{window.copiedReceipt=text;}}}));
 await page.locator('#rift-copy-receipt').click();
 expect(await page.evaluate(()=>window.copiedReceipt)).toContain('1,234 items');
 expect(await page.evaluate(()=>window.copiedReceipt)).toContain('Showing the most recent 2 of 1,234 banked items');
 for(const id of ['rift-campaign','rift-attempt-history'])if(!await page.locator('#'+id).evaluate(node=>node.open))await page.locator('#'+id+' > summary').click();
 const downloading=page.waitForEvent('download');await page.locator('#rift-export-records').click();const download=await downloading;
 const chunks=[];for await(const chunk of await download.createReadStream())chunks.push(chunk);
 expect(JSON.parse(Buffer.concat(chunks).toString('utf8')).career.gear).toBe(1234);
 const invalidTotals=await page.evaluate(async()=>{
  const data=await(await fetch('/api/abyss/rift')).json();
  return [-1,1,1.5,Number.MAX_SAFE_INTEGER+1].map(total=>{data.run.banked_items_total=total;try{RiftProtocol.validate(data,'GET');return false;}catch(_){return true;}});
 });expect(invalidTotals).toEqual([true,true,true,true]);
 const feedback=await page.evaluate(()=>{
  const before={id:'bounded',banked_gold:0,banked_items:['old','older'],banked_items_total:1234};
  const after={...before,banked_items:['older','new'],banked_items_total:1235};
  return [RiftLoot.confirmBank(before,after),RiftLoot.confirmBank(before,after),RiftLoot.confirmBank(after,after)];
 });expect(feedback).toEqual([true,false,false]);
 expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
});
