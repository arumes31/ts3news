const {test,expect}=require('@playwright/test');
test('Runesmith relic indicator distinguishes equipment from charged benefit',async({page})=>{
 await page.goto('/abyss/rift?subclass=runesmith&scenario=spawn-hazards');await expect(page.locator('#rift-start')).toBeEnabled();const {run}=await(await page.request.get('/api/abyss/rift')).json();const node=page.locator('#rift-relic-synergy');
 for(const [relic,resource,state] of [[false,2,'missing'],[true,0,'equipped'],[true,2,'armed']]){
  await page.evaluate(({run,relic,resource})=>{run.build.relic=relic;run.resource=resource;RiftHUD.update(run,false);},{run,relic,resource});await expect(node).toBeVisible();await expect(node).toHaveAttribute('data-state',state);await expect(node).toContainText('class barrier works with or without a relic');
 }
 await expect(node).toContainText('charged finisher damage ×1.15');await expect(node).toContainText('Paused');
 await page.setViewportSize({width:390,height:844});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);await node.scrollIntoViewIfNeeded();await node.screenshot({path:'test-results/relic-synergy-mobile.png'});
 await page.evaluate(run=>{run.build.class='oracle';RiftHUD.update(run,false);},run);await expect(node).toBeHidden();
});
