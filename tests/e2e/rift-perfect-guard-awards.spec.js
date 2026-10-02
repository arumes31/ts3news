const {test,expect}=require('@playwright/test');
test('perfect guard cosmetic badges use career totals at exact thresholds and exclude practice',async({page})=>{
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();const saved=(await(await page.request.get('/api/abyss/rift')).json()).run,awards=page.locator('#rift-perfect-guard-badges');
 for(const [total,count] of [[0,0],[9,0],[10,1],[99,1],[100,2],[999,2],[1000,3]]){
  await page.evaluate(({run,total})=>{run.past_expeditions={...run.past_expeditions,perfect_guards:Math.max(0,total-1)};run.stats.perfect_guards=total?1:0;RiftRecords.update(run);},{run:saved,total});await expect(awards.locator('span')).toHaveCount(count);
 }
 await expect(page.locator('#rift-perfect-guard-progress')).toContainText('All perfect-guard badges earned');await expect(page.locator('#rift-perfect-guard-awards')).toContainText('no rewards or combat bonuses');
 await page.setViewportSize({width:390,height:1600});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);await page.locator('#rift-perfect-guard-awards').scrollIntoViewIfNeeded();await page.evaluate(()=>window.scrollBy(0,-240));await page.locator('#rift-perfect-guard-awards').screenshot({path:'test-results/perfect-guard-badges.png'});
 await page.evaluate(run=>{run.practice={mode:'guard'};RiftRecords.update(run);},saved);await expect(page.locator('#rift-perfect-guard-awards')).toBeHidden();expect((await(await page.request.get('/api/abyss/rift')).json()).run).toEqual(saved);
});
