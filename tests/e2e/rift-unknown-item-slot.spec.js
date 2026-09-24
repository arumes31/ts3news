const {test,expect}=require('@playwright/test');
test('unknown gear slots remain visible through sorting comparison reload and receipt',async({page})=>{
 const name='Future-slot Discovery',slot='future_equipment_slot';
 await page.route('**/api/abyss/rift',async route=>{
  const response=await route.fetch(),data=await response.json();
  if(route.request().method()==='GET'&&data.run){
   const source=data.run.drops.find(d=>d.gear);if(!source)throw new Error('Checkpoint fixture needs gear');
   data.run.drops.push({...source,id:'unknown-slot',collected:true,banked:false,gear:{...source.gear,Name:name,Slot:slot,Stats:{STR:7}}});
   data.run.build.equipment={};
  }
  await route.fulfill({response,json:data});
 });
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
 const item=page.locator('#rift-loot > li').filter({has:page.locator('strong',{hasText:name})});
 for(const sort of ['found','rarity','slot']){
  await page.locator('#rift-loot-sort').selectOption(sort);await expect(item).toHaveCount(1);
  await item.locator(':scope > details > summary').click();await expect(item).toContainText(slot);await expect(item).toContainText('Ready to bank');
  await item.locator('.rift-gear-comparison > summary').click();await expect(item).toContainText('No item equipped in '+slot);await expect(item).toContainText('STR +7');
  await expect(item.locator('.rift-loot-icon')).toHaveCSS('background-position','33.3333% 66.6667%');
 }
 await page.reload();await expect(page.locator('#rift-start')).toBeEnabled();await expect(item).toHaveCount(1);
 const valid=await page.evaluate(async()=>{const data=await(await fetch('/api/abyss/rift')).json();try{window.RiftProtocol.validate(data,'GET');return true;}catch(_){return false;}});expect(valid).toBe(true);
 await page.evaluate(async({name})=>{
  const data=await(await fetch('/api/abyss/rift')).json(),run=data.run;
  run.drops.forEach(d=>d.banked=true);run.banked_items=[name];run.banked_loot=[{name,rarity:1,mission:1,tier:1}];run.banked_gold=10;
  window.RiftLoot.update(run);
 },{name});
 await page.locator('#rift-receipt > summary').click();await expect(page.locator('#rift-receipt-list > li')).toContainText(name);
 await page.setViewportSize({width:390,height:1000});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
});
