const {test,expect}=require('@playwright/test');
test('lantern damage persists and defending it preserves patrol rewards',async({page})=>{
 test.setTimeout(60000);await page.goto('/abyss/rift?scenario=lantern');await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('#rift-auto').uncheck();
 const saved=async()=>(await(await page.request.get('/api/abyss/rift')).json()).run;
 const progress=page.locator('#rift-room-objective-progress');
 await page.evaluate(()=>{window.lanternCues=[];const play=window.RiftAudio.play;window.RiftAudio.play=function(kind,...args){if(kind.startsWith('lantern_'))window.lanternCues.push(kind);return play.call(this,kind,...args);};});
 await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();await expect(progress).toContainText('Under threat');await expect.poll(async()=>(await saved()).room_objective.lantern.hp).toBeLessThanOrEqual(90);
 await page.locator('#rift-viewport').screenshot({path:'test-results/lantern-threat.png'});expect(await page.evaluate(()=>window.lanternCues.includes('lantern_threat')&&window.lanternCues.includes('lantern_hurt'))).toBe(true);
 await page.keyboard.press('Escape');await expect(page.locator('#rift-paused-badge')).toHaveText('Paused');const before=await saved();await page.evaluate(()=>history.replaceState(null,'','/abyss/rift'));await page.reload();await expect(page.locator('#rift-start')).toBeEnabled();expect((await saved()).room_objective).toEqual(before.room_objective);await page.locator('#rift-auto').uncheck();
 await page.evaluate(()=>{window.protectionCues=[];const play=window.RiftAudio.play;window.RiftAudio.play=function(kind,...args){if(kind==='lantern_protected')window.protectionCues.push(kind);return play.call(this,kind,...args);};});await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();
 await page.keyboard.down('KeyJ');try{await expect.poll(async()=>(await saved()).status,{timeout:10000}).toBe('cleared');}finally{await page.keyboard.up('KeyJ');}
 await expect(progress).toContainText('Protected');const result=await saved();expect(result.room_objective.lantern.hp).toBeGreaterThan(0);expect(result.stats.kills).toBe(1);expect(result.drops.length).toBe(1);expect(await page.evaluate(()=>window.protectionCues.length)).toBe(1);
 await page.locator('#rift-next').click();await expect.poll(async()=>(await saved()).room).toBe(1);await expect(progress).toContainText('Generators');
});
test('extinguished lantern fails expedition without changing character health',async({page})=>{
 await page.goto('/abyss/rift?scenario=lantern_failure');await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('#rift-auto').uncheck();const saved=async()=>(await(await page.request.get('/api/abyss/rift')).json()).run;const initial=await saved();
 await page.locator('#rift-start').click();await expect.poll(async()=>(await saved()).status,{timeout:10000}).toBe('defeated');await expect(page.locator('#rift-overlay-title')).toHaveText('The lantern went out.');await expect(page.locator('#rift-room-objective-progress')).toContainText('Lantern 0% · Extinguished');
 const result=await saved();expect(result.player.hp).toBe(initial.player.hp);expect(result.room_objective.complete).toBe(false);expect(result.gold).toBe(0);expect(result.drops).toHaveLength(0);await page.locator('#rift-viewport').screenshot({path:'test-results/lantern-extinguished.png'});
});
test('lantern snapshots validate health and geometry and mobile HUD fits',async({page})=>{
 await page.goto('/abyss/rift?scenario=lantern');await expect(page.locator('#rift-start')).toBeEnabled();const data=await(await page.request.get('/api/abyss/rift')).json();
 expect(await page.evaluate(data=>{const valid=v=>{try{window.RiftProtocol.validate(v,'GET');return true;}catch(_){return false;}};const health=structuredClone(data);health.run.room_objective.lantern.hp=101;const zone=structuredClone(data);zone.run.room_objective.zone.x+=20;const timer=structuredClone(data);timer.run.room_objective.seconds=1;const complete=structuredClone(data);complete.run.room_objective.complete=true;complete.run.room_objective.collected=1;complete.run.room_objective.lantern.hp=0;return [data,health,zone,timer,complete].map(valid);},data)).toEqual([true,false,false,false,false]);
 await page.setViewportSize({width:390,height:844});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
});
