const {test,expect}=require('@playwright/test');
test('attack and spell destruction persist without monster rewards',async({page})=>{
 test.setTimeout(90000);
 await page.goto('/abyss/rift?scenario=totems');await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('#rift-auto').uncheck();
 const progress=page.locator('#rift-room-objective-progress');await expect(progress).toContainText('Totems 0/3');
 const saved=async()=>(await(await page.request.get('/api/abyss/rift')).json()).run;
 await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();await page.locator('#rift-viewport').screenshot({path:'test-results/totem-room.png'});
 await page.keyboard.press('KeyJ');await expect.poll(async()=>{const r=await saved();const t=r.enemies.find(e=>e.kind==='totem');return t.hp<t.max_hp;}).toBe(true);
 await page.keyboard.press('Escape');await expect(page.locator('#rift-paused-badge')).toHaveText('Paused');const first=(await saved()).enemies.find(e=>e.kind==='totem');expect(first.hp).toBeGreaterThan(0);
 await page.evaluate(()=>history.replaceState(null,'','/abyss/rift'));await page.reload();await expect(page.locator('#rift-start')).toBeEnabled();expect((await saved()).enemies.find(e=>e.kind==='totem').hp).toBe(first.hp);await page.locator('#rift-auto').uncheck();
 await page.evaluate(()=>{window.totemCues=[];const play=window.RiftAudio.play;window.RiftAudio.play=function(kind,...args){if(kind.startsWith('totem_'))window.totemCues.push(kind);return play.call(this,kind,...args);};});
 await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();
 await page.keyboard.down('KeyJ');try{await expect(progress).toContainText('Totems 1/3');}finally{await page.keyboard.up('KeyJ');}
 const walkTo=async x=>{await page.keyboard.down('KeyD');try{await expect.poll(async()=>(await saved()).player.x,{intervals:[20]}).toBeGreaterThan(x);}finally{await page.keyboard.up('KeyD');}};
 await walkTo(375);
 const before=(await saved()).enemies.filter(e=>e.kind==='totem')[1].hp;await page.keyboard.press('Digit3');await expect.poll(async()=>(await saved()).enemies.filter(e=>e.kind==='totem')[1].hp).toBeLessThan(before);
 for(const x of [430,680]){
  await walkTo(x);
  await page.keyboard.down('KeyJ');try{await expect(progress).toContainText(x===430?'Totems 2/3':'Totems 3/3');}finally{await page.keyboard.up('KeyJ');}
 }
 await expect.poll(async()=>(await saved()).status).toBe('cleared');const result=await saved();expect(result.stats.kills).toBe(0);expect(result.drops.length).toBe(0);await expect.poll(()=>page.evaluate(()=>window.totemCues.filter(k=>k==='totem_break').length)).toBe(3);
 await page.locator('#rift-next').click();await expect.poll(async()=>(await saved()).room).toBe(2);await expect(page.locator('#rift-room-objective')).toBeHidden();
});
test('totem snapshots validate destruction and mobile HUD fits',async({page})=>{
 await page.goto('/abyss/rift?scenario=totems');await expect(page.locator('#rift-start')).toBeEnabled();const data=await(await page.request.get('/api/abyss/rift')).json();
 expect(await page.evaluate(data=>{const valid=v=>{try{window.RiftProtocol.validate(v,'GET');return true;}catch(_){return false;}};const count=structuredClone(data);count.run.room_objective.collected=1;const complete=structuredClone(data);complete.run.room_objective.complete=true;const missing=structuredClone(data);missing.run.enemies=missing.run.enemies.filter(e=>e.kind!=='totem');return [data,count,complete,missing].map(valid);},data)).toEqual([true,false,false,false]);
 await page.setViewportSize({width:390,height:844});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
});
