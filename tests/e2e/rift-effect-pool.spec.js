const {test,expect}=require('@playwright/test');

test('visual effects reuse bounded records without stale event fields',async({page})=>{
 await page.route('**/static/rift_renderer.js*',async route=>{
  const response=await route.fetch(),source=await response.text();
  const anchor='  window.RiftRenderer=renderer;';expect(source.split(anchor)).toHaveLength(2);
  await route.fulfill({response,body:source.replace(anchor,anchor+"\n window.effectRecords=()=>effects.slice();")});
 });
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
 const run=(await(await page.request.get('/api/abyss/rift')).json()).run;
 const result=await page.evaluate(run=>{
  const renderer=window.RiftRenderer,identities=new Set();
  run.paused=true;run.enemies=[];run.practice={};run.counter=0;run.events=[];
  renderer.snapshot(run,false);
  for(let i=1;i<=2000;i++){
   const event={id:i,kind:'hit',x:400,y:400,value:i};
   if(i<=40)event.oldPayload={encounter:'previous'};
   renderer.snapshot({...run,counter:i,events:[event]},false);
   window.effectRecords().forEach(record=>identities.add(record));
  }
  const active=window.effectRecords(),ids=active.map(e=>e.id),stale=active.some(e=>'oldPayload' in e);
  renderer.snapshot({...run,id:run.id+'-new',events:[]},false);
  return {allocated:identities.size,ids,stale,retiredClean:[...identities].every(e=>Object.keys(e).length===0),remaining:window.effectRecords().length};
 },run);
 expect(result.allocated).toBe(40);
 expect(result.ids).toEqual(Array.from({length:40},(_,i)=>1961+i));
 expect(result.stale).toBe(false);expect(result.retiredClean).toBe(true);expect(result.remaining).toBe(0);
});
