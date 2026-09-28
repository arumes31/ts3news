const {test,expect}=require('@playwright/test');
test('rest alcove holds waves through reload and Guard delays departure',async({page},info)=>{
 test.setTimeout(90000);const errors=[];page.on('pageerror',e=>errors.push(e.message));
 await page.goto('/abyss/rift?scenario=waves&condition=rest');await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('#rift-auto').uncheck();
 const saved=async()=>(await(await page.request.get('/api/abyss/rift')).json()).run;
 await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();
 await page.keyboard.down('KeyJ');try{await expect.poll(async()=>(await saved()).room_objective.next_wave_seconds).toBe(2.5);}finally{await page.keyboard.up('KeyJ');}
 await expect(page.locator('#rift-room-objective-progress')).toContainText('Resting');
 await page.waitForTimeout(3000);expect((await saved()).room_objective.wave).toBe(1);expect((await saved()).room_objective.next_wave_seconds).toBe(2.5);
 await page.locator('#rift-canvas').screenshot({path:info.outputPath('rest-alcove.png')});
 await page.keyboard.press('Escape');await expect(page.locator('#rift-paused-badge')).toHaveText('Paused');
 await page.evaluate(()=>history.replaceState(null,'','/abyss/rift'));await page.reload();await expect(page.locator('#rift-start')).toBeEnabled();expect((await saved()).room_objective.next_wave_seconds).toBe(2.5);await page.locator('#rift-auto').uncheck();await page.locator('#rift-start').click();
 await page.keyboard.down('KeyL');await page.keyboard.down('KeyD');try{await expect.poll(async()=>(await saved()).player.x,{timeout:12000}).toBeGreaterThan(300);}finally{await page.keyboard.up('KeyD');}
 await expect(page.locator('#rift-room-objective-progress')).toContainText('Reinforcements held');await page.waitForTimeout(3000);expect((await saved()).room_objective.next_wave_seconds).toBe(2.5);
 await page.keyboard.up('KeyL');await expect.poll(async()=>(await saved()).room_objective.wave,{timeout:10000}).toBe(2);expect(errors).toEqual([]);
});

test('rest geometry rejects invalid bounds and keeps legacy saves valid',async({page})=>{
 await page.goto('/abyss/rift?scenario=waves');await expect(page.locator('#rift-start')).toBeEnabled();const data=await(await page.request.get('/api/abyss/rift')).json();
 expect(await page.evaluate(data=>{
  const valid=v=>{try{RiftProtocol.validate(v,'GET');return true;}catch(_){return false;}};
  return [null,{x:70,y:335,w:210,h:140},{x:70,y:335,w:-1,h:140},{x:1550,y:335,w:210,h:140},{x:70,y:335,w:210,h:Infinity}].map(rest=>{const v=structuredClone(data);v.run.level.rooms[v.run.room].rest_alcove=rest;return valid(v);});
 },data)).toEqual([true,true,false,false,false]);
});
