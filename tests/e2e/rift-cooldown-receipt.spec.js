const {test,expect}=require('@playwright/test');
test('rewind receipt shows actual reductions, zero benefit, and protocol validation',async({page})=>{
 await page.goto('/abyss/rift?subclass=chronomancer&scenario=spawn-hazards');await expect(page.locator('#rift-start')).toBeEnabled();const data=await(await page.request.get('/api/abyss/rift')).json();const node=page.locator('#rift-cooldown-receipt');await expect(node).toBeHidden();
 const receipt={source:'Temporal Release',recovered:[{name:'Time Bolt',seconds:.4},{name:'Fireball',seconds:1.5},{name:'Jump',seconds:.7}]};
 await page.evaluate(({run,receipt})=>{run.last_cooldown_receipt=receipt;RiftHUD.update(run,false);},{run:data.run,receipt});await expect(node).toHaveText('Last rewind · Temporal Release: Time Bolt −0.4s, Fireball −1.5s, Jump −0.7s.');
 await page.setViewportSize({width:390,height:844});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);await node.scrollIntoViewIfNeeded();await node.screenshot({path:'test-results/cooldown-receipt-mobile.png'});
 await page.evaluate(run=>{run.last_cooldown_receipt={source:'Temporal Release',recovered:[]};RiftHUD.update(run,false);},data.run);await expect(node).toContainText('no active cooldowns to reduce');
 expect(await page.evaluate(({data,receipt})=>[undefined,receipt,{source:'Cast',recovered:[]},{source:'Cast',recovered:[{name:'Jump',seconds:2}]},{source:'Cast',recovered:[{name:'Jump',seconds:-1}]},{source:'Cast',recovered:null}].map(r=>{const v=structuredClone(data);v.run.last_cooldown_receipt=r;try{RiftProtocol.validate(v,'GET');return true;}catch(_){return false;}}),{data,receipt})).toEqual([true,true,true,false,false,false]);
});
