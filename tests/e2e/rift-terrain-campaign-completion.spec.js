const {test,expect}=require('@playwright/test');
const fs=require('node:fs');
const {createLedgeNavigator}=require('../../scripts/brawl-session-navigation.cjs');
// Navigation reads saved geometry; all movement, attacks and transitions use UI controls.
function terrainWaypoint(run,target){
 const p=run.player,arena=run.level.rooms[run.room];
 const gaps=(run.room_objective?.floor_segments||[]).filter(p=>p.collapsed);
 const gate=run.room_objective?.gate;if(gate?.closed)gaps.push(gate);
 for(const gap of gaps){
  const left=gap.x-24,right=gap.x+gap.w+24;
  const forward=p.x<right&&target.x>right,backward=p.x>left&&target.x<left;
  if((forward||backward)&&(p.y>350||target.y>350))return Math.abs(p.y-340)>5?{x:p.x,y:340}:{x:forward?right:left,y:340};
 }

 for(const b of arena.bridges||[]){
  if(Math.min(p.x,target.x)<=b.x+b.w+12&&Math.max(p.x,target.x)>=b.x-12){
   const y=b.y+b.h/2;
   if(Math.abs(p.y-y)>5)return {x:p.x,y};
   if(p.x<b.x+b.w+12&&target.x>b.x+b.w)return {x:b.x+b.w+24,y};
   if(p.x>b.x-12&&target.x<b.x)return {x:b.x-24,y};
  }
 }
 return target;
}
for(const mission of [4,2,5])test(`mission ${mission} completes terrain combat and all three tiers`,async({page},info)=>{
 test.setTimeout(420000);
 const report={mission,errors:[],expeditions:[]};
 page.on('pageerror',error=>report.errors.push({message:error.message}));
 const held=new Set();
 async function controls(wanted){
  for(const key of [...held])if(!wanted.has(key)){await page.keyboard.up(key);held.delete(key);}
  for(const key of wanted)if(!held.has(key)){await page.keyboard.down(key);held.add(key);}
 }
 const read=async()=>{const response=await page.request.get('/api/abyss/rift');expect(response.ok()).toBe(true);return(await response.json()).run;};
 async function expedition(index){
  const started=Date.now();const existing=await read();
  await page.locator(existing?.status==='complete'?'#rift-replay':'#rift-start').click();
  await expect(page.locator('#rift-overlay')).toBeHidden();
  let run;const tiers=[];let lastInputReset=0;const waypoint=createLedgeNavigator();
  try{
   while(Date.now()-started<360000){
    if(report.errors.length)throw Error('Runtime failure during campaign: '+report.errors[0].message);
    run=await read();expect(run.level.id).toBe(mission);if(run.room_objective?.floor_segments?.some(p=>p.collapsed))report.sawCollapsedFloor=true;report.lastCombat={room:run.room,status:run.status,player:{x:run.player.x,y:run.player.y,hp:run.player.hp},enemies:run.enemies.filter(e=>e.hp>0).map(e=>({kind:e.kind,x:e.x,y:e.y,hp:e.hp}))};expect(run.status,'combat must reach a checkpoint without death').not.toBe('defeated');
    if(run.status==='cleared'){
     await controls(new Set());tiers.push({room:run.room,clock:run.clock,kills:run.stats.kills});
     await expect(page.locator('#rift-next')).toBeVisible();await page.locator('#rift-next').click();
     const previous=run.room;
     await expect.poll(async()=>{const next=await read();return next.room!==previous||next.status==='complete';}).toBe(true);
     run=await read();if(run.status==='complete')break;
     await expect(page.locator('#rift-next')).toBeHidden();
     if(run.paused){await page.locator('#rift-start').click();}
     continue;
    }
    expect(run.status).toBe('fighting');
    const p=run.player;let target=run.enemies.filter(e=>e.hp>0).sort((a,b)=>Math.abs(a.x-p.x)-Math.abs(b.x-p.x))[0];
    const objective=run.room_objective;
    const capturing=!target&&objective&&!objective.complete&&objective.zone;
    if(capturing)target=objective.zone;
    const wanted=new Set(capturing?[]:['Space']);
    if(target){
     const dx=target.x-p.x,dy=target.y-p.y;
     const destination=terrainWaypoint(run,waypoint(run,target)),mx=destination.x-p.x,my=destination.y-p.y,detouring=destination!==target;
     if(Math.abs(my)>(detouring?5:10))wanted.add(my>0?'s':'w');
     if(Math.abs(mx)>(detouring||capturing?6:60)||!detouring&&Math.sign(dx)!==p.facing)wanted.add(mx>0?'d':'a');
     if(!capturing&&Math.abs(dx)<120&&Math.abs(dy)<30){
      const [builder,finisher]=run.build.signatures;
      if(run.resource>0&&!(run.skill_timers[finisher.id]>0)&&p.mana>=finisher.cost)wanted.add('e');
      else if(!(run.skill_timers[builder.id]>0)&&p.mana>=builder.cost)wanted.add('q');
      else wanted.add('j');
     }
    }
    // A tier snapshot can clear held client input; periodically release and
    // repress keys like a human adjusting movement, without bypassing controls.
    if(Date.now()-lastInputReset>1500){await controls(new Set());lastInputReset=Date.now();}
    await controls(wanted);await page.waitForTimeout(120);
   }
  }finally{await controls(new Set());}
  expect(run?.status).toBe('complete');expect(tiers.map(t=>t.room)).toEqual([0,1,2]);
  expect(run.stats.rooms_cleared).toBe(3);expect(run.stats.bosses).toBeGreaterThan(0);
  expect(run.banked_gold).toBeGreaterThan(0);
  await expect(page.locator('#rift-overlay')).toBeVisible();
  report.expeditions.push({index,id:run.id,replaySeed:run.replay_seed,tiers,status:run.status,durationMS:Date.now()-started,stats:run.stats,bankedGold:run.banked_gold,bankedItems:run.banked_items_total||0});
 }

 try{
  await page.goto('/abyss/rift?subclass=bloodblade&mission='+mission);
  await expect(page.locator('#rift-start')).toBeEnabled();
  await page.locator('#rift-auto').uncheck();
  await expedition(0);
  expect(report.errors).toEqual([]);
 }finally{fs.writeFileSync(info.outputPath('campaign-report.json'),JSON.stringify(report,null,2));}
});
