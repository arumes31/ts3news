const {test,expect}=require('@playwright/test');
test('collapse damage and progress survive reload before a WASD escape',async({page})=>{
 test.setTimeout(90000);
 await page.goto('/abyss/rift?scenario=collapse');await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('#rift-auto').uncheck();
 const saved=async()=>(await(await page.request.get('/api/abyss/rift')).json()).run;
 const progress=page.locator('#rift-room-objective-progress');const initial=await saved();
 await page.evaluate(()=>{window.collapseCues=[];const play=window.RiftAudio.play;window.RiftAudio.play=function(kind,...args){if(kind.startsWith('collapse_'))window.collapseCues.push(kind);return play.call(this,kind,...args);};});
 await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();await expect(progress).toContainText('Starts in');
 await expect.poll(async()=>(await saved()).player.hp,{timeout:10000}).toBeLessThan(initial.player.hp);
 await expect(progress).toContainText('Caught in the collapse');await page.locator('#rift-viewport').screenshot({path:'test-results/collapse-danger.png'});
 expect(await page.evaluate(()=>window.collapseCues.includes('collapse_start')&&window.collapseCues.includes('collapse_hit'))).toBe(true);
 await page.keyboard.press('Escape');await expect(page.locator('#rift-paused-badge')).toHaveText('Paused');const before=await saved();
 await page.evaluate(()=>history.replaceState(null,'','/abyss/rift'));await page.reload();await expect(page.locator('#rift-start')).toBeEnabled();const restored=await saved();
 expect(restored.room_objective).toEqual(before.room_objective);expect(restored.player.hp).toBe(before.player.hp);
 await page.locator('#rift-auto').uncheck();await page.evaluate(()=>{window.escapeCues=[];const play=window.RiftAudio.play;window.RiftAudio.play=function(kind,...args){if(kind==='collapse_escaped')window.escapeCues.push(kind);return play.call(this,kind,...args);};});await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();
 await page.keyboard.down('KeyW');try{await expect.poll(async()=>(await saved()).player.y,{intervals:[20]}).toBeLessThan(324);}finally{await page.keyboard.up('KeyW');}
 await page.keyboard.down('KeyD');try{await expect.poll(async()=>(await saved()).status,{timeout:18000,intervals:[30]}).toBe('cleared');}finally{await page.keyboard.up('KeyD');}
 await expect(progress).toContainText('Escaped');const result=await saved();expect(result.stats.kills).toBe(0);expect(result.drops).toHaveLength(0);expect(result.enemies.every(e=>e.hp===0&&e.pose==='escape')).toBe(true);expect(await page.evaluate(()=>window.escapeCues.length)).toBe(1);
 await page.locator('#rift-viewport').screenshot({path:'test-results/collapse-exit.png'});
 await page.locator('#rift-next').click();await expect.poll(async()=>(await saved()).room).toBe(1);await expect(progress).toContainText('Wave 1/3');
});
test('collapse validates trajectory and mobile HUD fits',async({page})=>{
 await page.goto('/abyss/rift?scenario=collapse');await expect(page.locator('#rift-start')).toBeEnabled();const data=await(await page.request.get('/api/abyss/rift')).json();
 expect(await page.evaluate(data=>{const valid=v=>{try{window.RiftProtocol.validate(v,'GET');return true;}catch(_){return false;}};const front=structuredClone(data);front.run.room_objective.collapse_x=700;const cooldown=structuredClone(data);cooldown.run.room_objective.collapse_hit_cooldown=2;const complete=structuredClone(data);complete.run.room_objective.complete=true;return [data,front,cooldown,complete].map(valid);},data)).toEqual([true,false,false,false]);
 await page.setViewportSize({width:390,height:844});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
});
