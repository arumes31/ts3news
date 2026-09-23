const {test,expect}=require('@playwright/test');
test('Oracle healing preview explains overflow and full-health charge gain',async({page})=>{
 await page.goto('/abyss/rift?subclass=oracle&scenario=spawn-hazards');await expect(page.locator('#rift-start')).toBeEnabled();const {run}=await(await page.request.get('/api/abyss/rift')).json();const node=page.locator('#rift-oracle-healing');
 for(const [hp,healing,overflow] of [[50,15,0],[95,5,10],[100,0,15]]){
  await page.evaluate(({run,hp})=>{run.player.max_hp=100;run.player.hp=hp;run.resource=0;RiftHUD.update(run,false);},{run,hp});await expect(node).toContainText(healing.toFixed(1)+' HP restored now; '+overflow.toFixed(1)+' HP overflow discarded (no barrier)');await expect(node).toContainText('grants 1 Grace charge even at full HP');
 }
 await page.evaluate(run=>{run.resource=3;RiftHUD.update(run,false);},run);await expect(node).toContainText('Charges full: no additional Grace charge');
 await page.setViewportSize({width:390,height:844});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);await node.scrollIntoViewIfNeeded();await node.screenshot({path:'test-results/oracle-overflow-mobile.png'});
 await page.evaluate(run=>{run.build.class='vanguard';RiftHUD.update(run,false);},run);await expect(node).toBeHidden();
});
