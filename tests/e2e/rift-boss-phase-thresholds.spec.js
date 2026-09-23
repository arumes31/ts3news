const {test,expect}=require('@playwright/test');
test('boss phase thresholds explain the current and next phase',async({page},testInfo)=>{
 await page.goto('/abyss/rift?scenario=boss-windup');await expect(page.locator('#rift-start')).toBeEnabled();
 const run=(await(await page.request.get('/api/abyss/rift')).json()).run;
 for(const [phase,label] of [[1,'Phase 1 · Next: phase 2 at 50% HP'],[2,'Phase 2 · Next: phase 3 at 25% HP'],[3,'Phase 3 · Final phase']]){
  await page.evaluate(({run,phase})=>{document.getElementById('rift-overlay').hidden=true;const boss=run.enemies.find(e=>e.kind==='boss');boss.phase=phase;boss.hp=boss.max_hp*(phase===1?1:phase===2?.5:.25);window.RiftHUD.update(run,true);},{run,phase});
  await expect(page.locator('#rift-boss-phase')).toHaveText(label);
  await expect(page.locator('#rift-boss-phase')).toBeVisible();
 }
 await page.locator('#rift-boss').screenshot({path:testInfo.outputPath('boss-thresholds.png')});
 await page.setViewportSize({width:390,height:844});
 expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
});
