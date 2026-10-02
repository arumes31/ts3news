const {test,expect}=require('@playwright/test');
test('victory captions name each defeated boss without suppressing the second',async({page})=>{
 await page.addInitScript(()=>localStorage.setItem('riftCaptions',JSON.stringify({version:1,enabled:true})));
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
 const run=(await(await page.request.get('/api/abyss/rift')).json()).run;
 await page.evaluate(run=>{
  run.paused=false;run.status='fighting';run.events=[];window.RiftFeedback.update(run,true,true);
  run.events=[{id:++run.counter,kind:'boss_death',actor_name:'Mossbound King',x:500},{id:++run.counter,kind:'boss_death',actor_name:'Void Archivist',x:550}];
  window.RiftFeedback.update(run,false,true);
 },run);
 await expect(page.locator('#rift-captions')).toContainText('Mossbound King defeated');
 await expect(page.locator('#rift-captions')).toContainText('Void Archivist defeated');
 await expect(page.locator('#rift-captions li')).toHaveCount(2);
});
