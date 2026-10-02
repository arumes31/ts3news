const {test,expect}=require('@playwright/test');
test('class rules stay separate from timed effects and barrier capacity',async({page})=>{
 await page.goto('/abyss/rift?subclass=berserker&scenario=spawn-hazards');await expect(page.locator('#rift-start')).toBeEnabled();const {run}=await(await page.request.get('/api/abyss/rift')).json();
 await page.evaluate(run=>{run.barrier=25;run.player.guard=true;run.skill_timers.connection_grace=1.2;run.skill_timers.perfect_guard=.22;run.skill_timers.slowed=1.4;run.slow_source='ice';RiftHUD.update(run,false);},run);
 await page.locator('#rift-effect-types > summary').click();await expect(page.locator('#rift-class-rules')).toContainText('Passive Fury');await expect(page.locator('#rift-class-rules')).toContainText(run.build.sequence);
 const effects=page.locator('#rift-temporary-effects');await expect(effects).toContainText('25 barrier remaining');await expect(effects).toContainText('no timer');await expect(effects).toContainText('connection recovery 1.2s');await expect(effects).toContainText('guard window: 0.22s');await expect(effects).toContainText('movement penalty');await expect(effects).not.toContainText('Fury');
 await page.setViewportSize({width:390,height:1600});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);await page.locator('#rift-effect-types').scrollIntoViewIfNeeded();await page.evaluate(()=>window.scrollBy(0,-240));await page.locator('#rift-effect-types').screenshot({path:'test-results/effect-types-mobile.png'});
 await page.evaluate(run=>{run.player.guard=false;run.skill_timers.perfect_guard=.22;RiftHUD.update(run,false);},run);await expect(effects).not.toContainText('guard window');
 await page.evaluate(run=>{run.barrier=0;run.skill_timers={};RiftHUD.update(run,false);},run);await expect(effects).toHaveText('No temporary protection or movement penalties active.');await expect(page.locator('#rift-class-rules')).toContainText('Passive Fury');
 expect((await(await page.request.get('/api/abyss/rift')).json()).run).toEqual(run);
});
