const {test,expect}=require('@playwright/test');
test('lost mark explanation names the confirmed cause and yields to a new target',async({page})=>{
 await page.goto('/abyss/rift?scenario=spawn-hazards');await expect(page.locator('#rift-start')).toBeEnabled();const {run}=await(await page.request.get('/api/abyss/rift')).json(),node=page.locator('#rift-mark-state');
 for(const reason of ['defeated','escaped']){await page.evaluate(({run,reason})=>{run.marked='';run.last_mark_end={target_name:'Marked scout',reason};RiftHUD.update(run,false);},{run,reason});await expect(node).toContainText(reason==='defeated'?'Marked scout was defeated':'Marked scout escaped');await expect(node).toContainText('Land a builder hit');}
 await page.setViewportSize({width:390,height:1200});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);await node.scrollIntoViewIfNeeded();await page.evaluate(()=>window.scrollBy(0,-240));await node.screenshot({path:'test-results/mark-end-mobile.png'});
 await page.evaluate(run=>{run.last_mark_end={target_name:'Old target',reason:'defeated'};run.marked=run.enemies.find(e=>e.hp>0).id;RiftHUD.update(run,false);},run);await expect(node).toHaveText('Marked: '+run.enemies.find(e=>e.hp>0).name);
 await page.evaluate(run=>{delete run.last_mark_end;run.marked='';RiftHUD.update(run,false);},run);await expect(node).toHaveText('No marked target');
 expect(await page.evaluate(async()=>{const data=await(await fetch('/api/abyss/rift')).json();data.run.last_mark_end={target_name:'Scout',reason:'guessed'};try{RiftProtocol.validate(data,'GET');return false;}catch(_){return true;}})).toBe(true);
});
