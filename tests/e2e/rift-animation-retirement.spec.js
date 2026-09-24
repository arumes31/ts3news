const {test,expect}=require('@playwright/test');

test('same-room replacements retire old death animations and reset revived actors',async({page})=>{
 await page.addInitScript(()=>{
  const set=Map.prototype.set;
  Map.prototype.set=function(key,value){if(key==='retired-probe-0'){window.deathAnimationMap=this;Map.prototype.set=set;}return set.call(this,key,value);};
 });
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
 const run=(await(await page.request.get('/api/abyss/rift')).json()).run;
 const result=await page.evaluate(run=>{
  const r=window.RiftRenderer;run.status='fighting';run.paused=true;run.events=[];
  for(let i=0;i<2000;i++){
   const next=structuredClone(run);next.counter=i+1;next.enemies=[{...run.player,id:'retired-probe-'+i,kind:'goblin',hp:0}];r.snapshot(next,true);
  }
  const retainedSize=window.deathAnimationMap.size,retained=[...window.deathAnimationMap.keys()].slice(-2);
  const live=structuredClone(run);live.counter=2001;live.enemies=[{...run.player,id:'retired-probe-1999',kind:'goblin',hp:10}];r.snapshot(live,true);
  const revivedSize=window.deathAnimationMap.size;
  live.enemies[0].hp=0;live.counter++;r.snapshot(structuredClone(live),false);
  const newDeathSize=window.deathAnimationMap.size;
  live.enemies=[];live.counter++;r.snapshot(structuredClone(live),true);
  return {retainedSize,retained,revivedSize,newDeathSize,clearedSize:window.deathAnimationMap.size};
 },run);
 expect(result.retainedSize).toBe(1);
 expect(result.retained).toEqual(['retired-probe-1999']);
 expect(result.revivedSize).toBe(0);expect(result.newDeathSize).toBe(1);expect(result.clearedSize).toBe(0);
});
