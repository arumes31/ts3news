const {test,expect}=require('@playwright/test');
test('Bloodblade receipt shows confirmed recovery and overflow with legacy support',async({page})=>{
 await page.goto('/abyss/rift?subclass=bloodblade&scenario=spawn-hazards');await expect(page.locator('#rift-start')).toBeEnabled();const data=await(await page.request.get('/api/abyss/rift')).json();const node=page.locator('#rift-blood-recovery');await expect(node).toBeHidden();
 const receipt={skill_id:'reap',skill_name:'Crimson Reap',healed:1,overflow:11};await page.evaluate(({run,receipt})=>{run.last_blood_recovery=receipt;RiftHUD.update(run,false);},{run:data.run,receipt});await expect(node).toHaveText('Last blood recovery · Crimson Reap: +1.0 HP; 11.0 HP overflow discarded.');
 await page.setViewportSize({width:390,height:844});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);await node.scrollIntoViewIfNeeded();await node.screenshot({path:'test-results/blood-recovery-mobile.png'});
 expect(await page.evaluate(({data,receipt})=>[undefined,receipt,{...receipt,healed:-1},{...receipt,overflow:-1},{...receipt,skill_name:null},null].map(r=>{const v=structuredClone(data);v.run.last_blood_recovery=r;try{RiftProtocol.validate(v,'GET');return true;}catch(_){return false;}}),{data,receipt})).toEqual([true,true,false,false,false,false]);
 await page.evaluate(({run,receipt})=>{run.last_blood_recovery={...receipt,healed:0,overflow:12};RiftHUD.update(run,false);},{run:data.run,receipt});await expect(node).toContainText('+0.0 HP; 12.0 HP overflow');
});
