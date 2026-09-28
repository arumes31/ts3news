const {test,expect}=require('@playwright/test');

test('volatile cluster pauses and reloads its warning, then damages once and drops loot',async({page},info)=>{
 test.setTimeout(90000);
 const errors=[];page.on('pageerror',e=>errors.push(e.message));
 await page.goto('/abyss/rift?scenario=volatile-cover');
 await expect(page.locator('#rift-start')).toBeEnabled();
 await page.locator('#rift-auto').uncheck();
 const saved=async()=>(await(await page.request.get('/api/abyss/rift')).json()).run;
 const cues=async()=>page.evaluate(()=>{window.terrainCues=[];const original=RiftAudio.play;RiftAudio.play=function(kind,...args){if(kind.startsWith('terrain_'))terrainCues.push(kind);return original.call(this,kind,...args);};});
 await cues();
 await page.locator('#rift-viewport').screenshot({path:info.outputPath('intact.png')});
 await page.locator('#rift-start').click();
 await page.keyboard.down('j');
 try{await expect.poll(async()=>(await saved()).level.rooms[0].cover.filter(c=>c.blast_fuse>0).length,{intervals:[20]}).toBe(3);}finally{await page.keyboard.up('j');}
 await page.keyboard.press('Escape');await expect(page.locator('#rift-paused-badge')).toHaveText('Paused');
 const paused=await saved();expect(paused.level.rooms[0].cover.every(c=>c.blast_fuse>0)).toBe(true);
 expect(await page.evaluate(()=>terrainCues.filter(c=>c==='terrain_arming').length)).toBe(3);
 await page.waitForTimeout(1400);expect((await saved()).level.rooms[0].cover).toEqual(paused.level.rooms[0].cover);
 await page.locator('#rift-viewport').screenshot({path:info.outputPath('warning-paused.png')});
 await page.setViewportSize({width:390,height:844});await page.emulateMedia({reducedMotion:'reduce'});
 await page.locator('#rift-viewport').screenshot({path:info.outputPath('warning-mobile-reduced.png')});
 expect((await saved()).level.rooms[0].cover).toEqual(paused.level.rooms[0].cover);
 await page.setViewportSize({width:1280,height:720});
 await page.evaluate(()=>history.replaceState(null,'','/abyss/rift'));await page.reload();await expect(page.locator('#rift-start')).toBeEnabled();
 expect((await saved()).level.rooms[0].cover).toEqual(paused.level.rooms[0].cover);
 await cues();await page.locator('#rift-auto').uncheck();await page.locator('#rift-start').click();
 await expect.poll(async()=>(await saved()).level.rooms[0].cover.every(c=>!c.blast_fuse)).toBe(true);
 const after=await saved();expect(after.enemies.find(e=>e.id==='blast-target').hp).toBe(88);expect(after.enemies.find(e=>e.id==='blast-loot').hp).toBe(0);expect(after.stats.kills).toBe(1);expect(after.drops).toHaveLength(1);
 expect(await page.evaluate(()=>terrainCues.filter(c=>c==='terrain_arming').length)).toBe(0);expect(await page.evaluate(()=>terrainCues.filter(c=>c==='terrain_blast').length)).toBe(3);
 await page.waitForTimeout(700);expect((await saved()).enemies.find(e=>e.id==='blast-target').hp).toBe(88);
 await page.locator('#rift-viewport').screenshot({path:info.outputPath('spent.png')});expect(errors).toEqual([]);
});
