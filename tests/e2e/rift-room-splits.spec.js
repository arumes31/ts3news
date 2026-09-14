const {test,expect}=require('@playwright/test');

test('room splits distinguish recorded zero from missing tiers and validate values',async({page})=>{
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('#rift-attempt-history > summary').click();await expect(page.locator('#rift-room-splits')).toContainText('Tier 1 unavailable');
 const validation=await page.evaluate(async()=>{const data=await(await fetch('/api/abyss/rift')).json();const results=[[0,12.5,null],[-1,0,0],[1,2],[1,2,'3'],undefined].map(splits=>{data.run.room_splits=splits;try{window.RiftProtocol.validate(data,'GET');return true;}catch(_){return false;}});data.run.room_splits=[0,12.5,null];data.run.attempt_history=[{mission:1,outcome:'exited',at_ms:100000,class:'vanguard',seconds:12.5,hp:100,max_hp:100,splits:[0,12.5,null]}];window.RiftRecords.update(data.run);return results;});
 expect(validation).toEqual([true,false,false,false,true]);await expect(page.locator('#rift-room-splits')).toContainText('Tier 1 0.0s · Tier 2 12.5s · Tier 3 unavailable');await expect(page.locator('#rift-attempt-list')).toContainText('Tier 1 0.0s · Tier 2 12.5s · Tier 3 unavailable');await expect(page.locator('#rift-attempt-list')).toContainText('Damaging hits unavailable');
 await page.setViewportSize({width:390,height:844});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
});
