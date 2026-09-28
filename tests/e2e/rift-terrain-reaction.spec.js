const {test,expect}=require('@playwright/test');

test('volatile cluster pauses and reloads its warning, then damages once and drops loot',async({page},info)=>{
 test.setTimeout(90000);
 const errors=[];page.on('pageerror',e=>errors.push(e.message));
 await page.goto('/abyss/rift?scenario=volatile-cover');
 await expect(page.locator('#rift-start')).toHaveText('Resume expedition');
 await page.locator('#rift-auto').uncheck();
 const saved=async()=>(await(await page.request.get('/api/abyss/rift')).json()).run;
 const cues=async()=>page.evaluate(()=>{window.terrainCues=[];const original=RiftAudio.play;RiftAudio.play=function(kind,...args){if(kind.startsWith('terrain_'))terrainCues.push(kind);return original.call(this,kind,...args);};});
 await cues();
 await page.locator('#rift-viewport').screenshot({path:info.outputPath('intact.png')});
 // Hide only the pause panel in this additional render inspection; retain the actual UI capture above.
 await page.locator('#rift-viewport').screenshot({path:info.outputPath('intact-world.png'),style:'#rift-overlay { visibility: hidden !important; }'});
 await page.locator('#rift-start').click();
 await page.keyboard.down('j');
 try{await expect.poll(async()=>(await saved()).level.rooms[0].cover.filter(c=>c.blast_fuse>0).length,{intervals:[20]}).toBe(3);}finally{await page.keyboard.up('j');}
 await page.keyboard.press('Escape');await expect(page.locator('#rift-paused-badge')).toHaveText('Paused');
 const paused=await saved();expect(paused.level.rooms[0].cover.every(c=>c.blast_fuse>0)).toBe(true);
 expect(await page.evaluate(()=>terrainCues.filter(c=>c==='terrain_arming').length)).toBe(3);
 await page.waitForTimeout(1400);expect((await saved()).level.rooms[0].cover).toEqual(paused.level.rooms[0].cover);
 await page.locator('#rift-viewport').screenshot({path:info.outputPath('warning-paused.png')});
 // Hide only the pause panel in this additional render inspection; retain the actual UI capture above.
 await page.locator('#rift-viewport').screenshot({path:info.outputPath('warning-paused-world.png'),style:'#rift-overlay { visibility: hidden !important; }'});
 await page.setViewportSize({width:390,height:844});await page.emulateMedia({reducedMotion:'reduce'});
 await page.locator('#rift-viewport').screenshot({path:info.outputPath('warning-mobile-reduced.png')});
 // Hide only the pause panel in this additional render inspection; retain the actual UI capture above.
 await page.locator('#rift-viewport').screenshot({path:info.outputPath('warning-mobile-reduced-world.png'),style:'#rift-overlay { visibility: hidden !important; }'});
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

test('volatile artwork is deferred until a mission needs it and reused across tiers',async({page})=>{
 const requests=[];page.on('request',request=>{if(new URL(request.url()).pathname.endsWith('/rift_volatile_cover.png'))requests.push(request.url());});
 await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();expect(requests).toHaveLength(0);
 await page.evaluate(()=>RiftRenderer.prepareRun({room:0,level:{region:0,rooms:[{cover:[{material:'wood'}]}]}}));expect(requests).toHaveLength(0);
 await page.evaluate(()=>RiftRenderer.prepareRun({room:0,level:{region:0,rooms:[{}, {cover:[{material:'wood',volatile:true}]}]}}));expect(requests).toHaveLength(1);
 await page.evaluate(()=>RiftRenderer.prepareRun({room:1,level:{region:0,rooms:[{}, {cover:[{material:'wood',volatile:true}]}]}}));expect(requests).toHaveLength(1);
});

test('failed volatile artwork recovers and its decode gates saved combat',async({page})=>{
 let fail=true,attempts=0;
 await page.route('**/static/rift_volatile_cover.png*',route=>{attempts++;return fail?route.abort():route.continue();});
 await page.addInitScript(()=>{
  const decode=HTMLImageElement.prototype.decode;
  window.volatileDecodeGate=new Promise(resolve=>window.releaseVolatileDecode=resolve);
  HTMLImageElement.prototype.decode=async function(){await decode.call(this);if(this.src.includes('rift_volatile_cover.png')){window.volatileDecoding=true;await window.volatileDecodeGate;}};
 });
 await page.goto('/abyss/rift?scenario=volatile-cover');
 await expect(page.locator('#rift-start')).toHaveText('Reload artwork');expect(attempts).toBe(1);
 await expect(page.locator('#rift-status')).toContainText('Could not load volatile crate artwork');
 const before=(await(await page.request.get('/api/abyss/rift')).json()).run;
 await page.evaluate(()=>history.replaceState(null,'','/abyss/rift'));
 fail=false;await page.locator('#rift-start').click();
 await expect.poll(()=>page.evaluate(()=>!!window.volatileDecoding)).toBe(true);
 await expect(page.locator('#rift-start')).toBeDisabled();
 const pending=(await(await page.request.get('/api/abyss/rift')).json()).run;
 expect(pending.id).toBe(before.id);expect(pending.clock).toBe(before.clock);expect(pending.level.rooms[0].cover).toEqual(before.level.rooms[0].cover);
 await page.evaluate(()=>window.releaseVolatileDecode());await expect(page.locator('#rift-start')).toBeEnabled();
 await expect(page.locator('#rift-start')).not.toHaveText('Reload artwork');expect(attempts).toBe(2);
});

test('authored mission eleven previews and renders its three marked crates',async({page},info)=>{
 const errors=[];page.on('pageerror',error=>errors.push(error.message));
 await page.goto('/abyss/rift?mission=11');await expect(page.locator('#rift-start')).toBeEnabled();
 await page.locator('#rift-mission-preview > summary').click();
 await expect(page.locator('#rift-room-previews [data-terrain="wood"] title')).toHaveText(Array(3).fill('Volatile cover: breaking it ignites nearby crates after 1.2 seconds'));
 await page.locator('#rift-auto').uncheck();await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();
 await page.keyboard.press('Escape');await expect(page.locator('#rift-paused-badge')).toHaveText('Paused');
 const run=(await(await page.request.get('/api/abyss/rift')).json()).run;
 expect(run.level.id).toBe(11);expect(run.level.rooms[0].cover.filter(cover=>cover.volatile)).toHaveLength(3);
 await page.locator('#rift-viewport').screenshot({path:info.outputPath('mission-11-crates.png')});
 // Hide only the pause panel in this additional render inspection; retain the actual UI capture above.
 await page.locator('#rift-viewport').screenshot({path:info.outputPath('mission-11-crates-world.png'),style:'#rift-overlay { visibility: hidden !important; }'});expect(errors).toEqual([]);
});
