const {test,expect}=require('@playwright/test');
async function clearWaves(page){
 await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();
 await page.keyboard.down('KeyJ');try{await expect.poll(async()=>(await(await page.request.get('/api/abyss/rift')).json()).run.status,{timeout:30000}).toBe('cleared');}finally{await page.keyboard.up('KeyJ');}
}
test('chest opens once after confirmed loot, loads after Start and survives reload open',async({page},info)=>{
 test.setTimeout(90000);const requests=[],errors=[];page.on('request',r=>{if(r.url().includes('rift_loot_chest.png'))requests.push(r.url());});page.on('pageerror',e=>errors.push(e.message));
 await page.goto('/abyss/rift?scenario=waves');await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('#rift-auto').uncheck();expect(requests).toEqual([]);
 await page.evaluate(()=>{window.chestOpenings=[];window.chestSounds=0;const observe=RiftChest.observe,play=RiftAudio.play;RiftChest.observe=function(run,replay,now){const opened=observe(run,replay,now);if(opened)chestOpenings.push({frame:RiftChest.frame(now,false).index,drops:run.drops.length});return opened;};RiftAudio.play=function(kind,...args){if(kind==='chest_open')chestSounds++;return play.call(this,kind,...args);};});
 await clearWaves(page);await page.evaluate(()=>RiftChest.ready());await expect.poll(()=>page.evaluate(()=>!!RiftChest.image())).toBe(true);expect(requests).toHaveLength(1);
 expect(await page.evaluate(()=>chestOpenings)).toEqual([{frame:0,drops:expect.any(Number)}]);expect(await page.evaluate(()=>chestSounds)).toBe(1);
 await expect(page.locator('#rift-chest-preview')).toBeVisible();
 await page.waitForTimeout(900);await page.locator('#rift-canvas').screenshot({path:info.outputPath('loot-chest-open.png')});
 const desktop=page.viewportSize();await page.setViewportSize({width:390,height:844});
 await expect.poll(async()=>page.evaluate(()=>{const panel=document.getElementById('rift-room-actions').getBoundingClientRect(),stage=document.querySelector('.rift-viewport').getBoundingClientRect();return panel.top>=stage.top&&panel.bottom<=stage.bottom&&document.documentElement.scrollWidth<=innerWidth;})).toBe(true);
 await page.locator('.rift-viewport').screenshot({path:info.outputPath('loot-chest-mobile.png')});await page.setViewportSize(desktop);
 const before=(await(await page.request.get('/api/abyss/rift')).json()).run;
 await page.keyboard.press('Escape');await page.evaluate(()=>history.replaceState(null,'','/abyss/rift'));await page.reload();await expect(page.locator('#rift-start')).toBeEnabled();await page.evaluate(()=>RiftChest.ready());
 expect(await page.evaluate(()=>RiftChest.frame(0,false).index)).toBe(5);
 const after=(await(await page.request.get('/api/abyss/rift')).json()).run;expect(after.drops).toEqual(before.drops);expect(after.gold).toBe(before.gold);expect(errors).toEqual([]);
});
test('missing optional chest art does not prevent tier banking',async({page})=>{
 test.setTimeout(90000);await page.route('**/rift_loot_chest.png*',route=>route.abort());
 await page.goto('/abyss/rift?scenario=waves');await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('#rift-auto').uncheck();await clearWaves(page);await page.evaluate(()=>RiftChest.ready());expect(await page.evaluate(()=>RiftChest.image())).toBeNull();
 await page.locator('#rift-next').click();await expect.poll(async()=>(await(await page.request.get('/api/abyss/rift')).json()).run.room).toBe(2);
});

test('missing optional chest script leaves combat and banking usable',async({page})=>{
 test.setTimeout(90000);const errors=[];page.on('pageerror',e=>errors.push(e.message));await page.route('**/rift_chest.js*',route=>route.abort());
 await page.goto('/abyss/rift?scenario=waves');await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('#rift-auto').uncheck();await clearWaves(page);
 await page.locator('#rift-next').click();await expect.poll(async()=>(await(await page.request.get('/api/abyss/rift')).json()).run.room).toBe(2);expect(errors).toEqual([]);
});
