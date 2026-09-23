const {test,expect}=require('@playwright/test');
const classes={vanguard:'perfect guard',berserker:'execution damage',marksman:'precision piercing',beastmaster:'equipped pets',elementalist:'reaction',chronomancer:'rewind',oracle:'at full health',geomancer:'armor piercing',bloodblade:'restores health',voidwalker:'costs health',runesmith:'without a relic',alchemist:'mixture charges'};
for(const [subclass,tip] of Object.entries(classes))test(subclass+' low-resource hint names the equipped builder and spending choice',async({page})=>{
 await page.goto('/abyss/rift?subclass='+subclass+'&scenario=spawn-hazards');await expect(page.locator('#rift-start')).toBeEnabled();const {run}=await(await page.request.get('/api/abyss/rift')).json(),builder=run.build.signatures.find(s=>s.role==='builder'),node=page.locator('#rift-low-resource');
 await page.evaluate(run=>{run.resource=0;RiftHUD.update(run,false);},run);await expect(node).toContainText(builder.name+' (Q)');await expect(node).toContainText(tip);await expect(node).toContainText('Builder: Paused');await expect(node).toContainText('0/3');
 await page.evaluate(run=>{run.resource=1;RiftHUD.update(run,false);},run);await expect(node).toContainText('One charge already enables');
 await page.evaluate(run=>{run.resource=2;RiftHUD.update(run,false);},run);await expect(node).toBeHidden();
});
test('low-resource hint respects key remapping, builder order, missing skills and ended runs',async({page})=>{
 await page.goto('/abyss/rift?scenario=spawn-hazards');await expect(page.locator('#rift-start')).toBeEnabled();const {run}=await(await page.request.get('/api/abyss/rift')).json(),node=page.locator('#rift-low-resource');
 await page.locator('#rift-controls-open').click();await page.locator('[data-remap="signature1"]').click();await page.keyboard.press('t');await page.locator('#rift-controls-close').click();
 await page.evaluate(run=>{run.build.signatures.reverse();run.resource=0;run.paused=false;run.skill_timers[run.build.signatures.find(s=>s.role==='builder').id]=2;RiftHUD.update(run,true);},run);await expect(node).toContainText('(T)');await expect(node).toContainText('2.0 seconds cooldown');
 await page.setViewportSize({width:390,height:1200});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);await node.scrollIntoViewIfNeeded();await page.evaluate(()=>window.scrollBy(0,-240));await node.screenshot({path:'test-results/low-resource-mobile.png'});
 await page.evaluate(run=>{run.resource=0;run.build.signatures=[];RiftHUD.update(run,false);},run);await expect(node).toContainText('Unlock and equip');
 await page.evaluate(run=>{run.resource=0;run.status='complete';RiftHUD.update(run,false);},run);await expect(node).toBeHidden();
});
