const {test,expect}=require('@playwright/test');
test('ritual interrupts persist, pulses warn, and channelers award normal loot',async({page})=>{
 test.setTimeout(90000);
 await page.goto('/abyss/rift?scenario=ritual');await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('#rift-auto').uncheck();
 const saved=async()=>(await(await page.request.get('/api/abyss/rift')).json()).run;
 const progress=page.locator('#rift-room-objective-progress');
 const hook=()=>page.evaluate(()=>{window.ritualCues=[];const play=window.RiftAudio.play;window.RiftAudio.play=function(kind,...args){if(kind.startsWith('ritual_'))window.ritualCues.push(kind);return play.call(this,kind,...args);};});
 await hook();await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();
 await expect.poll(async()=>(await saved()).room_objective.channels[0].seconds).toBeGreaterThan(2);
 await page.keyboard.press('KeyJ');await expect.poll(()=>page.evaluate(()=>window.ritualCues.includes('ritual_interrupt'))).toBe(true);
 expect((await saved()).room_objective.channels[0].seconds).toBeLessThan(1);
 await expect.poll(async()=>(await saved()).room_objective.channels[0].seconds).toBeGreaterThan(1);
 await page.keyboard.press('Escape');await expect(page.locator('#rift-paused-badge')).toHaveText('Paused');const before=await saved();
 await page.evaluate(()=>history.replaceState(null,'','/abyss/rift'));await page.reload();await expect(page.locator('#rift-start')).toBeEnabled();
 const restored=await saved();expect(restored.room_objective.channels).toEqual(before.room_objective.channels);expect(restored.enemies[0].hp).toBe(before.enemies[0].hp);
 await page.locator('#rift-auto').uncheck();await hook();await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();
 await expect.poll(async()=>(await saved()).room_objective.channels[0].seconds,{timeout:12000}).toBeGreaterThan(6);
 await page.locator('#rift-viewport').screenshot({path:'test-results/ritual-warning.png'});
 await expect.poll(async()=>(await saved()).player.hp,{timeout:6000}).toBeLessThan(before.player.hp);
 expect(await page.evaluate(()=>window.ritualCues.includes('ritual_warning')&&window.ritualCues.includes('ritual_pulse'))).toBe(true);
 const ids=(await saved()).room_objective.channels.map(c=>c.enemy_id);
 for(const id of ids){
  const target=(await saved()).enemies.find(e=>e.id===id);
  if(target.x-40>(await saved()).player.x){await page.keyboard.down('KeyD');try{await expect.poll(async()=>(await saved()).player.x,{intervals:[20],timeout:10000}).toBeGreaterThan(target.x-45);}finally{await page.keyboard.up('KeyD');}}
  await page.keyboard.down('KeyJ');try{await expect.poll(async()=>(await saved()).enemies.find(e=>e.id===id).hp,{timeout:10000}).toBe(0);}finally{await page.keyboard.up('KeyJ');}
 }
 await expect.poll(async()=>(await saved()).status).toBe('cleared');await expect(progress).toContainText('Ritual 3/3 · Interrupted');
 const result=await saved();expect(result.stats.kills).toBe(3);expect(result.drops.length).toBe(3);expect(await page.evaluate(()=>window.ritualCues.filter(k=>k==='ritual_complete').length)).toBe(1);
 await page.locator('#rift-next').click();await expect.poll(async()=>(await saved()).room).toBe(2);await expect(page.locator('#rift-room-objective')).toBeHidden();
});
test('ritual snapshots validate channel links and mobile HUD fits',async({page})=>{
 await page.goto('/abyss/rift?scenario=ritual');await expect(page.locator('#rift-start')).toBeEnabled();const data=await(await page.request.get('/api/abyss/rift')).json();
 expect(await page.evaluate(data=>{const valid=v=>{try{window.RiftProtocol.validate(v,'GET');return true;}catch(_){return false;}};const timer=structuredClone(data);timer.run.room_objective.channels[0].seconds=8;const duplicate=structuredClone(data);duplicate.run.room_objective.channels[1].enemy_id=duplicate.run.room_objective.channels[0].enemy_id;const missing=structuredClone(data);missing.run.room_objective.channels[0].enemy_id='missing';const count=structuredClone(data);count.run.room_objective.collected=1;return [data,timer,duplicate,missing,count].map(valid);},data)).toEqual([true,false,false,false,false]);
 await page.setViewportSize({width:390,height:844});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
});
