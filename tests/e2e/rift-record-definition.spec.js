const {test,expect}=require('@playwright/test');
test('attempt comparisons require matching known mission definitions',async({page})=>{
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
 const data=await(await page.request.get('/api/abyss/rift')).json();
 const show=async(definitions)=>page.evaluate(({data,definitions})=>{data.run.attempt_history=definitions.map((definition,i)=>({mission:1,difficulty:'Wayfarer',definition,outcome:'completed',at_ms:1700000000000,class:'vanguard',seconds:10+i,hp:50,max_hp:100,hits:0,splits:[3,3,4]}));RiftProtocol.validate(data,'GET');RiftRecords.update(data.run);},{data,definitions});
 const a='level-v1:'+'a'.repeat(64),b='level-v1:'+'b'.repeat(64),comparison=page.locator('#rift-attempt-comparison');
 await show([a,b]);await expect(comparison).toContainText('Different or unknown mission versions are not compared');await expect(page.locator('#rift-attempt-list > li')).toHaveCount(2);
 await show([a,b,a]);await expect(comparison).toContainText('combat time +2.0s');
 await show([undefined,undefined]);await expect(comparison).toContainText('Mission version unavailable; comparison unavailable');await expect(page.locator('#rift-attempt-list')).toContainText('Mission version unavailable');
 const rejected=await page.evaluate(data=>{const invalid=[null,12,'', 'level-v1:xyz','level-v2:'+'a'.repeat(64)];return invalid.every(definition=>{data.run.mission_definition=definition;try{RiftProtocol.validate(data,'GET');return false;}catch(_){return true;}});},data);expect(rejected).toBe(true);
 expect((await(await page.request.get('/api/abyss/rift')).json()).run).toEqual(data.run);
});
