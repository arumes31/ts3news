const {test,expect}=require('@playwright/test');

// Read closure sizes in the test response only; production code has no debug API.
async function observeAnimationState(page){
 await page.route('**/static/rift_renderer.js*',async route=>{
  const response=await route.fetch(),source=await response.text();
  const anchor='  window.RiftRenderer=renderer;';expect(source.split(anchor)).toHaveLength(2);
  await route.fulfill({response,body:source.replace(anchor,anchor+"\n  window.animationState=()=>({effects:effects.length,decals:decals.length,deaths:deaths.size,animationTime,previous:previous?.id||null});")});
 });
}
for(const reduced of [false,true])test('bounded effects expire and transitions retire state: reduced '+reduced,async({page})=>{
 const errors=[];page.on('pageerror',error=>errors.push(error.message));await observeAnimationState(page);
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
 const run=(await(await page.request.get('/api/abyss/rift')).json()).run;
 await page.evaluate(({run,reduced})=>{
  const r=window.RiftRenderer;r.reduced=reduced;run.status='fighting';run.paused=true;run.enemies=[];
  for(let i=0;i<2000;i++){
   const next=structuredClone(run);next.counter=i+1;next.events=[{id:i+1,kind:'fire',x:400,y:400,value:1}];r.snapshot(next,false);
  }
  run.counter=2000;run.events=[];window.animationRun=run;
 },{run,reduced});
 const paused=await page.evaluate(()=>window.animationState());expect(paused.effects).toBe(40);expect(paused.decals).toBe(40);
 await page.waitForTimeout(250);expect(await page.evaluate(()=>window.animationState())).toEqual(paused);
 await page.evaluate(()=>{window.animationRun.paused=false;window.RiftRenderer.snapshot(window.animationRun,true);});
 await expect.poll(()=>page.evaluate(()=>window.animationState().effects)).toBe(0);
 await expect.poll(()=>page.evaluate(()=>window.animationState().decals)).toBe(0);
 const transitions=await page.evaluate(()=>{
  const results=[],r=window.RiftRenderer;let run=structuredClone(window.animationRun);run.paused=true;
  for(const change of ['room','mission','run','rewind']){
   run.counter+=2;run.events=[{id:run.counter,kind:'fire',x:400,y:400,value:1}];run.enemies=[{...run.player,id:'dead-'+change,hp:0}];r.snapshot(structuredClone(run),false);
   const before=window.animationState();run.events=[];run.enemies=[];
   if(change==='room')run.room=(run.room+1)%3;
   if(change==='mission')run.level.id++;
   if(change==='run')run.id+='-next';
   if(change==='rewind')run.counter=0;
   r.snapshot(structuredClone(run),true);results.push({change,before,after:window.animationState()});
  }
  return results;
 });
 for(const {before,after} of transitions){expect(before.effects).toBeGreaterThan(0);expect(before.decals).toBeGreaterThan(0);expect(before.deaths).toBe(1);expect(after.effects).toBe(0);expect(after.decals).toBe(0);expect(after.deaths).toBe(0);expect(after.previous).toBeNull();}
 expect(errors).toEqual([]);
});


test('shared artwork caches evict old identities and bound backdrop variants',async({page})=>{
 await page.route('**/static/abyss_combat_art.js*',async route=>{
  const response=await route.fetch(),source=await response.text(),anchor='})(window);';expect(source.split(anchor)).toHaveLength(2);
  await route.fulfill({response,body:source.replace(anchor,"global.artCacheState=()=>Object.fromEntries(Object.entries({effects,profiles,actorProfiles,backdropCache}).map(([name,cache])=>[name,{size:cache.size,first:String(cache.keys().next().value)}]));\n"+anchor)});
 });
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
 const result=await page.evaluate(()=>{
  const art=window.AbyssCombatArt;
  for(let i=0;i<5000;i++){
   const profile=art.profileFor({id:'cache-probe-'+i,name:'Cache probe '+i,kind:'fire'});
   if(i<1000){art.actorProfile({art_key:'monster:cache-probe-'+i,name:'Cache probe '+i});art.effectFrame(profile,'impact',i);}
   art.backdrop({biome:'Unknown biome '+i,depth:i});
  }
  const before=window.artCacheState();
  const actor=art.actorProfile({art_key:'monster:cache-probe-0',name:'Cache probe 0'});
  const profile=art.profileFor({id:'cache-probe-0',name:'Cache probe 0',kind:'fire'});
  const regenerated=art.effectFrame(profile,'impact',0);
  return {before,after:window.artCacheState(),identity:actor.identity,regenerated:regenerated.startsWith('data:image/svg+xml,')};
 });
 expect(result.before.profiles.size).toBe(4096);expect(result.before.actorProfiles.size).toBe(512);expect(result.before.effects.size).toBe(256);expect(result.before.backdropCache.size).toBeLessThanOrEqual(13);
 for(const key of ['profiles','actorProfiles','effects']){expect(result.before[key].first).not.toContain('cache-probe-0');expect(result.after[key].size).toBe(result.before[key].size);}
 expect(result.identity).toBe('monster:cache-probe-0');expect(result.regenerated).toBe(true);
});
