const {test,expect}=require('@playwright/test');

test('live monster roster and class poses stay inside actor atlases',async({page})=>{
 const pageErrors=[];page.on('pageerror',error=>pageErrors.push(error.message));
 await page.goto('/abyss/rift?scenario=checkpoint');
 await expect(page.locator('#rift-start')).toBeEnabled();
 const data=await(await page.request.get('/api/abyss/rift')).json();
 expect(data.bestiary.length).toBeGreaterThan(0);
 const result=await page.evaluate(async({roster,player})=>{
  const art=window.AbyssCombatArt,renderer=window.RiftRenderer;
  // Direct actor probes must prepare both lazily loaded hero sheets first.
  await Promise.all(['vanguard','berserker','marksman','beastmaster','elementalist','chronomancer','oracle','geomancer','bloodblade','voidwalker','runesmith','alchemist'].map(className=>renderer.prepareBuild({class:className})));
  const errors=[],assets=new Set(),rigs=new Set();let frames=0,draws=0;
  const names={scribe:'Scribe Without Eyes',remembers:'Abyss That Remembers'};
  const probes=art.rigs.map(rig=>({name:names[rig]||rig,art_key:'bounds-probe:'+rig,kind:'goblin',element:'physical'}));
  const actors=[...roster,...probes],sharedCells=new Set();
  for(const unit of actors){
   rigs.add(window.RiftBestiary.profile(unit).rig);
   for(const pose of Object.keys(art.poses))for(const index of [0,1,2,3,4,5,6,7,1000000]){
    const frame=window.RiftBestiary.frame(unit,pose,index),s=frame.source;
    frames++;assets.add(frame.asset);sharedCells.add(frame.asset+JSON.stringify(s));
    if(!s||![s.x,s.y,s.width,s.height].every(Number.isFinite)||s.x<0||s.y<0||s.width<=0||s.height<=0||s.x+s.width>1.000001||s.y+s.height>1.000001)errors.push({name:unit.name,pose,index,source:s});
   }
  }
  const ctx=document.getElementById('rift-canvas').getContext('2d'),original=ctx.drawImage;
  let current='';
  ctx.drawImage=function(img,...args){
   if(args.length!==8)return;
   draws++;assets.add(img.src||'cached-frame');
   const [x,y,w,h]=args;
   if(![x,y,w,h].every(Number.isFinite)||x<0||y<0||w<=0||h<=0||x+w>img.width+.001||y+h>img.height+.001)errors.push({current,source:args.slice(0,4),size:[img.width,img.height]});
  };
  const classes=['vanguard','berserker','marksman','beastmaster','elementalist','chronomancer','oracle','geomancer','bloodblade','voidwalker','runesmith','alchemist'];
  const units=[...actors,...classes.map(kind=>({...player,id:'player',kind,art_key:undefined})),...['goblin','archer','knight','boss','wolf','spore'].map(kind=>({kind,id:'legacy-'+kind}))];
  await renderer.prepareCreatures(units);
  const poses=['idle','run','attack','cast','windup','guard_walk','land','recovery','hit','knockdown','stagger','ultimate_anticipation','victory','defeat'];
  const reduced=renderer.reduced;
  try{
   for(const still of [false,true]){
    renderer.reduced=still;
    for(const unit of units)for(const pose of poses)for(const time of [0,.2,.3]){
     current=(unit.name||unit.kind)+' / '+pose+' / '+time+' / '+still;
     renderer.renderActor({...unit,id:unit.id||'probe',x:450,y:410,hp:pose==='defeat'?0:100,max_hp:100,facing:-1,pose,pose_time:time},performance.now());
    }
   }
  }finally{ctx.drawImage=original;renderer.reduced=reduced;}
  return {errors:errors.slice(0,20),frames,draws,assets:[...assets],rigs:[...rigs],expectedRigs:art.rigs,sharedCells:sharedCells.size,actors:actors.length,units:units.length};
 },{roster:data.bestiary,player:data.run.player});
 expect(pageErrors).toEqual([]);
 expect(result.errors).toEqual([]);
 expect(result.frames).toBe(result.actors*5*9);
 expect(result.rigs.sort()).toEqual(result.expectedRigs.slice().sort());
 expect(result.sharedCells).toBe(32*8);
 expect(result.draws).toBeGreaterThanOrEqual(result.units*14*3*2);
 expect(result.assets.length).toBeGreaterThanOrEqual(4);
 console.log('Actor atlas coverage',JSON.stringify({monsters:data.bestiary.length,frames:result.frames,draws:result.draws,rigs:result.rigs.length}));
});
