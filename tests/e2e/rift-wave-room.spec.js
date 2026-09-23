const {test,expect}=require('@playwright/test');
test('three waves preserve countdown, corpses and tier loot',async({page})=>{
 test.setTimeout(90000);
 await page.goto('/abyss/rift?scenario=waves');await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('#rift-auto').uncheck();
 const saved=async()=>(await(await page.request.get('/api/abyss/rift')).json()).run;
 const progress=page.locator('#rift-room-objective-progress');await expect(progress).toContainText('Wave 1/3');
 await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();
 await page.keyboard.down('KeyJ');try{await expect(progress).toContainText('Reinforcements');}finally{await page.keyboard.up('KeyJ');}
 await page.keyboard.press('Escape');await expect(page.locator('#rift-paused-badge')).toHaveText('Paused');const before=await saved();expect(before.room_objective.wave).toBe(1);expect(before.status).toBe('fighting');
 await page.evaluate(()=>history.replaceState(null,'','/abyss/rift'));await page.reload();await expect(page.locator('#rift-start')).toBeEnabled();expect((await saved()).room_objective.next_wave_seconds).toBe(before.room_objective.next_wave_seconds);await page.locator('#rift-auto').uncheck();
 await page.evaluate(()=>{window.waveCues=[];const play=window.RiftAudio.play;window.RiftAudio.play=function(kind,...args){if(kind.startsWith('wave'))window.waveCues.push(kind);return play.call(this,kind,...args);};});
 await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();await expect(progress).toContainText('Wave 2/3');
 await page.keyboard.down('KeyJ');try{await expect.poll(async()=>(await saved()).status,{timeout:25000}).toBe('cleared');}finally{await page.keyboard.up('KeyJ');}
 const after=await saved();expect(after.room_objective.complete).toBe(true);expect(after.enemies.length).toBe(after.room_objective.waves.flat().length);expect(after.enemies.every(e=>e.hp===0)).toBe(true);expect(new Set(after.drops.map(d=>d.id)).size).toBe(after.drops.length);expect(after.drops.length).toBeGreaterThan(0);
 await expect(progress).toContainText('Wave 3/3 · Survived');await expect.poll(()=>page.evaluate(()=>window.waveCues.filter(k=>k==='waves_complete').length)).toBe(1);expect(await page.evaluate(()=>window.waveCues.filter(k=>k==='wave_start').length)).toBe(2);
 await page.locator('#rift-next').click();await expect.poll(async()=>(await saved()).room).toBe(2);await expect(page.locator('#rift-room-objective')).toBeHidden();
});
test('wave protocol and mobile layout',async({page})=>{
 await page.goto('/abyss/rift?scenario=waves');await expect(page.locator('#rift-start')).toBeEnabled();const data=await(await page.request.get('/api/abyss/rift')).json();
 expect(await page.evaluate(data=>{const valid=v=>{try{window.RiftProtocol.validate(v,'GET');return true;}catch(_){return false;}};const count=structuredClone(data);count.run.room_objective.wave=4;const timer=structuredClone(data);timer.run.room_objective.next_wave_seconds=3;const duplicate=structuredClone(data);duplicate.run.room_objective.waves[1][0].id=duplicate.run.room_objective.waves[0][0].id;return [data,count,timer,duplicate].map(valid);},data)).toEqual([true,false,false,false]);
 await page.setViewportSize({width:390,height:844});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
});

test('final wave gives tier-clear instructions',async({page})=>{
 await page.route('**/api/abyss/rift',async route=>{const response=await route.fetch();const data=await response.json();if(route.request().method()==='GET'&&data.run?.room_objective?.kind==='survive_waves'){data.run.room_objective.wave=3;await route.fulfill({response,json:data});}else await route.fulfill({response});});
 await page.goto('/abyss/rift?scenario=waves');await expect(page.locator('#rift-start')).toBeEnabled();await expect(page.locator('#rift-room-objective-help')).toContainText('Defeat the final group');
});
