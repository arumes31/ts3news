const {test,expect}=require('@playwright/test');
test('queued abilities expire exactly at the grace boundary, including blocked recovery',async({page})=>{
 await page.goto('/abyss/rift?scenario=spawn-hazards');await expect(page.locator('#rift-start')).toBeEnabled();const saved=(await(await page.request.get('/api/abyss/rift')).json()).run;
 const result=await page.evaluate(saved=>{
  const run=structuredClone(saved),intent=RiftIntents,original=Object.getOwnPropertyDescriptor(performance,'now');let clock=1000;Object.defineProperty(performance,'now',{configurable:true,value:()=>clock});
  run.player.cooldown=0;run.player.mana=100;run.skill_timers={};
  const take=()=>intent.take(run,()=>false,false).skill;
  try{
   intent.sync(run,true);intent.press('signature0');clock+=1199;const inside=take();
   intent.press('signature0');clock+=1200;const boundary=take();
   intent.press('signature0');run.player.mana=0;const blocked=take();clock+=1201;run.player.mana=100;const recovered=take();
   intent.press('signature0');const fresh=take();return {inside,boundary,blocked,recovered,fresh,id:run.build.signatures[0].id};
  }finally{intent.reset();if(original)Object.defineProperty(performance,'now',original);else delete performance.now;}
 },saved);
 expect(result.inside).toBe(result.id);expect(result.fresh).toBe(result.id);expect(result.boundary).toBe('');expect(result.blocked).toBe('');expect(result.recovered).toBe('');expect((await(await page.request.get('/api/abyss/rift')).json()).run).toEqual(saved);
});
test('room, mission and run changes discard queued casts while same-room updates retain fresh taps',async({page})=>{
 await page.goto('/abyss/rift?scenario=spawn-hazards');await expect(page.locator('#rift-start')).toBeEnabled();const saved=(await(await page.request.get('/api/abyss/rift')).json()).run;
 const results=await page.evaluate(saved=>{
  const results=[];for(const change of ['same','room','mission','run','replay']){
   const run=structuredClone(saved);run.player.cooldown=0;run.player.mana=100;run.skill_timers={};RiftIntents.sync(run,true);RiftIntents.press('signature0');RiftIntents.press('skill0');
   if(change==='room')run.room++;if(change==='mission')run.level.id++;if(change==='run')run.id+='-new';RiftIntents.sync(run,change==='replay');
   const casts=[RiftIntents.take(run,()=>false,false).skill,RiftIntents.take(run,()=>false,false).skill];RiftIntents.press('signature0');const fresh=RiftIntents.take(run,()=>false,false).skill;results.push({change,casts,fresh});
  }RiftIntents.reset();return results;
 },saved);
 for(const result of results){expect(result.casts).toEqual(result.change==='same'?[saved.build.signatures[0].id,saved.build.skills[0].id]:['','']);expect(result.fresh).toBe(saved.build.signatures[0].id);}
 expect((await(await page.request.get('/api/abyss/rift')).json()).run).toEqual(saved);
});

test('a skill tapped during a pending tier advance cannot fire in the new room',async({page})=>{
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();let release,observed;const held=new Promise(resolve=>release=resolve),pending=new Promise(resolve=>observed=resolve),casts=[];
 await page.route('**/api/abyss/rift',async route=>{if(route.request().method()==='POST'){const body=route.request().postDataJSON();if(body.kind==='advance'){observed();await held;}if(body.kind==='step'&&body.input.skill)casts.push(body.input.skill);}await route.continue();});
 await page.locator('#rift-start').click();await pending;await page.keyboard.press('Digit1');release();const read=async()=>(await(await page.request.get('/api/abyss/rift')).json()).run;
 await expect.poll(async()=>(await read()).room).toBe(1);await page.waitForTimeout(400);expect(casts).toEqual([]);await page.keyboard.press('Digit1');await expect.poll(()=>casts.length).toBe(1);await page.keyboard.press('Escape');
});
