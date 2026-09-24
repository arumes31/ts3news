const {test,expect}=require('@playwright/test');
test('Alchemist mixture sequence follows charges, targets and remapped keys',async({page})=>{
 await page.goto('/abyss/rift?subclass=alchemist&scenario=spawn-hazards');await expect(page.locator('#rift-start')).toBeEnabled();const {run}=await(await page.request.get('/api/abyss/rift')).json();const node=page.locator('#rift-alchemist-sequence');
 for(const charges of [0,1,2,3]){
  await page.evaluate(({run,charges})=>{run.resource=charges;run.marked=charges?run.enemies.find(e=>e.hp>0).id:'';RiftHUD.update(run,false);},{run,charges});await expect(node).toContainText(run.build.resource+' '+charges+'/3');await expect(node).toContainText(charges===0?'Start with Volatile Mixture (Q)':charges===3?run.build.resource+' full: spend with Catalytic Burst (E)':'Build again with Volatile Mixture (Q) or spend with Catalytic Burst (E)');
  if(charges){await expect(node).toContainText((charges*3)+'% maximum HP');await expect(node).toContainText('35 percentage points');}else await expect(node).toContainText('Land a builder hit');
 }
 await page.locator('#rift-controls-open').click();await page.locator('[data-remap="signature1"]').click();await page.keyboard.press('t');await page.locator('#rift-controls-close').click();await page.evaluate(run=>{run.resource=3;RiftHUD.update(run,false);},run);await expect(node).toContainText('Catalytic Burst (T)');
 await page.setViewportSize({width:390,height:844});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);await node.scrollIntoViewIfNeeded();await node.screenshot({path:'test-results/alchemist-sequence-mobile.png'});
 await page.evaluate(run=>{run.build.signatures=[];RiftHUD.update(run,false);},run);await expect(node).toContainText('equip both');
 await page.evaluate(run=>{run.build.class='oracle';RiftHUD.update(run,false);},run);await expect(node).toBeHidden();
});
