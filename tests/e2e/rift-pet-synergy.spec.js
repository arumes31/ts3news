const {test,expect}=require('@playwright/test');
test('Beastmaster pet bonus displays saved count, cap and charge gate',async({page})=>{
 await page.goto('/abyss/rift?subclass=beastmaster&scenario=spawn-hazards');await expect(page.locator('#rift-start')).toBeEnabled();const {run}=await(await page.request.get('/api/abyss/rift')).json(),node=page.locator('#rift-pet-synergy');
 for(const pets of [0,1,2,3,5])for(const charges of [0,1]){
  await page.evaluate(({run,pets,charges})=>{run.build.pets=pets;run.resource=charges;RiftHUD.update(run,false);},{run,pets,charges});await expect(node).toContainText('Pets in this expedition: '+pets+' · '+Math.min(3,pets)+'/3');await expect(node).toHaveAttribute('data-state',pets===0?'none':charges?'armed':'building');if(pets)await expect(node).toContainText('×'+(1+Math.min(3,pets)*.1).toFixed(1));else await expect(node).toContainText('No pet damage bonus');
 }
 await expect(node).toContainText('Builders and empty finishers get no pet multiplier');await expect(node).toContainText('Finisher: Paused');await page.setViewportSize({width:390,height:1200});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);await node.scrollIntoViewIfNeeded();await page.evaluate(()=>window.scrollBy(0,-240));await node.screenshot({path:'test-results/pet-synergy-mobile.png'});
 await page.evaluate(run=>{run.build.pets=2;run.build.signatures=[];RiftHUD.update(run,false);},run);await expect(node).toHaveAttribute('data-state','missing');await expect(node).toContainText('Equip a class finisher');
 await page.evaluate(run=>{run.build.class='oracle';RiftHUD.update(run,false);},run);await expect(node).toBeHidden();
});
