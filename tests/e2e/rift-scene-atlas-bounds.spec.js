const {test,expect}=require('@playwright/test');

test('campaign scenery loot effects and victory atlas rectangles stay in bounds',async({page})=>{
 const pageErrors=[];page.on('pageerror',error=>pageErrors.push(error.message));
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
 const data=await(await page.request.get('/api/abyss/rift')).json();expect(data.levels).toHaveLength(100);
 const result=await page.evaluate(async data=>{
  const r=window.RiftRenderer,ctx=document.getElementById('rift-canvas').getContext('2d'),draw=ctx.drawImage;
  await Promise.all(['vanguard','bloodblade'].map(className=>r.prepareBuild({class:className})));
  const errors=[],cells={},victories=[];let label='',draws=0,scenes=0;
  ctx.drawImage=function(img,...a){
   if(a.length===8){
    draws++;const [x,y,w,h]=a,key=img.src?new URL(img.src,location.href).pathname:'cached-frame';
    if(![x,y,w,h].every(Number.isFinite)||x<0||y<0||w<=0||h<=0||x+w>img.width+.001||y+h>img.height+.001)errors.push({label,key,source:a.slice(0,4),size:[img.width,img.height]});
    (cells[key]??=new Set()).add([x,y,w,h].join(','));
   }
   return draw.call(this,img,...a);
  };
  const waitFrame=async()=>{const before=r.frameCount;while(r.frameCount===before)await new Promise(requestAnimationFrame);};
  const base=structuredClone(data.run);base.paused=true;base.status='fighting';base.enemies=[];base.events=[];base.room_objective=null;
  base.drops=['weapon','offhand','ranged','head','chest','feet','hands','ring','neck','relic','future'].map((Slot,i)=>({id:'atlas-'+i,x:200+i*45,y:430,gear:{Slot,Name:Slot,Rarity:1}}));
  base.drops.push({id:'gold',x:750,y:430,gold:1});
  try{
   for(const level of data.levels)for(let room=0;room<level.rooms.length;room++){
    const run=structuredClone(base);run.level=structuredClone(level);run.room=room;
    const arena=run.level.rooms[room];arena.cover=[{material:'wood',hp:100,max_hp:100},{material:'wood',hp:40,max_hp:100},{material:'wood',hp:0,max_hp:100},{material:'stone',hp:100,max_hp:100}].map((c,i)=>({...c,x:220+i*140,y:340,w:60,h:40}));
    label='mission '+level.id+' room '+room;r.snapshot(run,true);await waitFrame();scenes++;
   }
   for(const kind of ['vanguard','berserker','marksman','beastmaster','elementalist','chronomancer','oracle','geomancer','bloodblade','voidwalker','runesmith','alchemist']){
    const run=structuredClone(base);run.build.class=kind;run.status='complete';run.player.kind=kind;run.player.pose='victory';run.player.pose_time=4;
    label='victory '+kind;r.snapshot(run,true);await waitFrame();victories.push(r.lastVictoryPose?.subclass);
   }
   const preview=document.createElement('canvas');preview.width=320;preview.height=160;preview.getContext('2d').drawImage=ctx.drawImage;
   for(const kind of ['vanguard','berserker','marksman','beastmaster','elementalist','chronomancer','oracle','geomancer','bloodblade','voidwalker','runesmith','alchemist','warrior','ranger','arcanist','warden','reaver','artificer']){
    for(const skill of ['slash','fire','ice','shield','void','rare_item','arrow'])for(const elapsed of [-1,0,125,250,375,500,625,750,10000]){
     label='skill preview '+kind+' '+skill+' '+elapsed;r.drawSkillPreview(preview,{kind:skill},{class:kind},elapsed);
    }
    for(const state of [{jump:.3},{guard:true},{knockdown:.5}]){
     label='player state '+kind;r.renderActor({...base.player,kind,pose:'idle',pose_time:0,...state},performance.now());
    }
   }
   const run=structuredClone(base);run.id='atlas-effects';run.paused=false;run.events=['slash','fire','ice','shield','void','rare_item'].map((kind,i)=>({id:i+1,kind,x:300+i*60,y:400,value:1}));run.counter=6;
   label='effect frames';r.snapshot(run,false);const until=performance.now()+1050;while(performance.now()<until)await waitFrame();
  }finally{ctx.drawImage=draw;}
  return {errors:errors.slice(0,20),scenes,draws,victories,cells:Object.fromEntries(Object.entries(cells).map(([key,value])=>[key,[...value]]))};
 },data);
 expect(pageErrors).toEqual([]);
 expect(result.errors).toEqual([]);expect(result.scenes).toBe(300);expect(new Set(result.victories).size).toBe(12);
 expect(result.cells['/static/rift_regions.png'].length).toBe(10);
 expect(result.cells['/static/rift_terrain_cover.png'].length).toBe(4);
 expect(result.cells['/static/rift_items.png'].length).toBe(12);
 expect(result.cells['/static/rift_props.png'].length).toBe(8);
 expect(result.cells['/static/rift_effects.png'].length).toBe(36);
 console.log('Scene atlas coverage',JSON.stringify({scenes:result.scenes,draws:result.draws,cells:Object.fromEntries(Object.entries(result.cells).map(([key,value])=>[key,value.length]))}));
});
