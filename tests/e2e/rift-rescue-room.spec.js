const {test,expect}=require('@playwright/test');
test('cage damage persists and attack/spell rescues award no monster loot',async({page})=>{
 test.setTimeout(90000);
 await page.goto('/abyss/rift?scenario=rescue');await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('#rift-auto').uncheck();
 const saved=async()=>(await(await page.request.get('/api/abyss/rift')).json()).run;
 const progress=page.locator('#rift-room-objective-progress');await expect(progress).toContainText('Companions 0/2');
 await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();await page.locator('#rift-viewport').screenshot({path:'test-results/rescue-caged.png'});
 await page.keyboard.press('KeyJ');await expect.poll(async()=>{const e=(await saved()).enemies.find(e=>e.kind==='cage');return e.hp<e.max_hp;}).toBe(true);
 await page.keyboard.press('Escape');await expect(page.locator('#rift-paused-badge')).toHaveText('Paused');const before=await saved();
 await page.evaluate(()=>history.replaceState(null,'','/abyss/rift'));await page.reload();await expect(page.locator('#rift-start')).toBeEnabled();const restored=await saved();expect(restored.room_objective).toEqual(before.room_objective);expect(restored.enemies.filter(e=>e.kind==='cage').map(e=>e.hp)).toEqual(before.enemies.filter(e=>e.kind==='cage').map(e=>e.hp));
 await page.locator('#rift-auto').uncheck();await page.evaluate(()=>{window.rescueCues=[];const play=window.RiftAudio.play;window.RiftAudio.play=function(kind,...args){if(kind==='companion_freed'||kind==='rescue_complete')window.rescueCues.push(kind);return play.call(this,kind,...args);};});await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();
 await page.keyboard.down('KeyJ');try{await expect(progress).toContainText('Companions 1/2');}finally{await page.keyboard.up('KeyJ');}
 await page.locator('#rift-viewport').screenshot({path:'test-results/rescue-freed.png'});
 await page.keyboard.press('Escape');await expect(page.locator('#rift-paused-badge')).toHaveText('Paused');const freed=await saved();await page.reload();await expect(page.locator('#rift-start')).toBeEnabled();expect((await saved()).room_objective.captives[0]).toEqual(freed.room_objective.captives[0]);await page.locator('#rift-auto').uncheck();
 await page.evaluate(()=>{window.finalRescueCues=[];const play=window.RiftAudio.play;window.RiftAudio.play=function(kind,...args){if(kind==='companion_freed'||kind==='rescue_complete')window.finalRescueCues.push(kind);return play.call(this,kind,...args);};});await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();
 const walk=async(key,axis,target)=>{await page.keyboard.down(key);try{await expect.poll(async()=>(await saved()).player[axis],{intervals:[20],timeout:10000}).toBeGreaterThan(target);}finally{await page.keyboard.up(key);}};
 await walk('KeyS','y',476);await walk('KeyD','x',910);
 const second=(await saved()).enemies.filter(e=>e.kind==='cage')[1];await page.keyboard.press('Digit3');await expect.poll(async()=>(await saved()).enemies.find(e=>e.id===second.id).hp).toBeLessThan(second.hp);
 if((await saved()).status!=='cleared'){await walk('KeyD','x',1080);await page.keyboard.down('KeyJ');try{await expect.poll(async()=>(await saved()).status).toBe('cleared');}finally{await page.keyboard.up('KeyJ');}}
 await expect(progress).toContainText('Companions 2/2');const result=await saved();expect(result.stats.kills).toBe(0);expect(result.drops).toHaveLength(0);expect(result.room_objective.captives.every(c=>c.freed)).toBe(true);expect(await page.evaluate(()=>window.finalRescueCues.filter(k=>k==='companion_freed').length)).toBe(1);expect(await page.evaluate(()=>window.finalRescueCues.filter(k=>k==='rescue_complete').length)).toBe(1);
 await page.locator('#rift-next').click();await expect.poll(async()=>(await saved()).room).toBe(1);await expect(progress).toContainText('Relic');
});
test('rescue rejects inconsistent captive links and fits mobile',async({page})=>{
 await page.goto('/abyss/rift?scenario=rescue');await expect(page.locator('#rift-start')).toBeEnabled();const data=await(await page.request.get('/api/abyss/rift')).json();
 expect(await page.evaluate(data=>{const valid=v=>{try{window.RiftProtocol.validate(v,'GET');return true;}catch(_){return false;}};const position=structuredClone(data);position.run.room_objective.captives[0].x+=10;const duplicate=structuredClone(data);duplicate.run.room_objective.captives[1].cage_id=duplicate.run.room_objective.captives[0].cage_id;const freed=structuredClone(data);freed.run.room_objective.captives[0].freed=true;const future=structuredClone(data);future.run.room_objective.captives[0].freed_at=100;return [data,position,duplicate,freed,future].map(valid);},data)).toEqual([true,false,false,false,false]);
 await page.setViewportSize({width:390,height:844});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
});
