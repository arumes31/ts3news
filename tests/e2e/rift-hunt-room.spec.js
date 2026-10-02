const {test,expect}=require('@playwright/test');
test('saved hunt targets secure the room and survivors retreat without rewards',async({page})=>{
 test.setTimeout(90000);
 await page.goto('/abyss/rift?scenario=hunt');await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('#rift-auto').uncheck();
 const saved=async()=>(await(await page.request.get('/api/abyss/rift')).json()).run;
 const progress=page.locator('#rift-room-objective-progress');await expect(progress).toContainText('Targets 0/3');const targets=(await saved()).room_objective.targets;
 await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();await page.locator('#rift-viewport').screenshot({path:'test-results/hunt-targets.png'});
 await page.keyboard.press('KeyJ');await expect(progress).toContainText('Targets 1/3');await page.keyboard.press('Escape');await expect(page.locator('#rift-paused-badge')).toHaveText('Paused');
 await page.evaluate(()=>history.replaceState(null,'','/abyss/rift'));await page.reload();await expect(page.locator('#rift-start')).toBeEnabled();expect((await saved()).room_objective.targets).toEqual(targets);await expect(progress).toContainText('Targets 1/3');await page.locator('#rift-auto').uncheck();
 await page.evaluate(()=>{window.huntCues=[];const play=window.RiftAudio.play;window.RiftAudio.play=function(kind,...args){if(kind==='hunt_complete')window.huntCues.push(kind);return play.call(this,kind,...args);};});
 await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();
 for(const x of [430,680]){
  await page.keyboard.down('KeyD');try{await expect.poll(async()=>(await saved()).player.x,{intervals:[20]}).toBeGreaterThan(x);}finally{await page.keyboard.up('KeyD');}
  await page.keyboard.press('KeyJ');await expect(progress).toContainText(x===430?'Targets 2/3':'Targets 3/3');
 }
 await expect.poll(async()=>(await saved()).status).toBe('cleared');const result=await saved();expect(result.stats.kills).toBe(3);expect(result.drops.length).toBe(3);const survivors=result.enemies.filter(e=>!targets.includes(e.id));expect(survivors.length).toBeGreaterThan(0);expect(survivors.every(e=>e.hp===0&&e.pose==='escape')).toBe(true);await expect.poll(()=>page.evaluate(()=>window.huntCues.length)).toBe(1);
 await page.locator('#rift-next').click();await expect.poll(async()=>(await saved()).room).toBe(2);await expect(page.locator('#rift-room-objective')).toBeHidden();
});
test('hunt protocol rejects missing and duplicate targets and mobile HUD fits',async({page})=>{
 await page.goto('/abyss/rift?scenario=hunt');await expect(page.locator('#rift-start')).toBeEnabled();const data=await(await page.request.get('/api/abyss/rift')).json();
 expect(await page.evaluate(data=>{const valid=v=>{try{window.RiftProtocol.validate(v,'GET');return true;}catch(_){return false;}};const missing=structuredClone(data);missing.run.room_objective.targets[0]='missing';const duplicate=structuredClone(data);duplicate.run.room_objective.targets[0]=duplicate.run.room_objective.targets[1];const count=structuredClone(data);count.run.room_objective.collected=1;const fleeing=structuredClone(data);fleeing.run.enemies.find(e=>e.id===fleeing.run.room_objective.targets[0]).kind='treasure';return [data,missing,duplicate,count,fleeing].map(valid);},data)).toEqual([true,false,false,false,false]);
 await page.setViewportSize({width:390,height:844});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
});
