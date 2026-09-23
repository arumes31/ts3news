const {test,expect}=require('@playwright/test');
test('carry, reload and deliver relic through real arena geometry',async({page})=>{
 test.setTimeout(90000);
 await page.goto('/abyss/rift?scenario=relic');await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('#rift-auto').uncheck();
 const saved=async()=>(await(await page.request.get('/api/abyss/rift')).json()).run;
 const progress=page.locator('#rift-room-objective-progress');await expect(progress).toContainText('Find the relic');
 const listen=()=>page.evaluate(()=>{window.relicCues=[];const play=window.RiftAudio.play;window.RiftAudio.play=function(kind,...args){if(kind.startsWith('relic_'))window.relicCues.push(kind);return play.call(this,kind,...args);};});
 await listen();await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();
 await page.keyboard.down('KeyD');try{await expect.poll(async()=>(await saved()).room_objective.carrying,{intervals:[20]}).toBe(true);}finally{await page.keyboard.up('KeyD');}
 await expect(progress).toContainText('Carrying · 30% slower');expect(await page.evaluate(()=>window.relicCues.filter(k=>k==='relic_pickup').length)).toBe(1);await page.locator('#rift-viewport').screenshot({path:'test-results/relic-carrying.png'});
 await page.keyboard.press('Escape');await expect(page.locator('#rift-paused-badge')).toHaveText('Paused');await expect(progress).toContainText('30% slower · Paused');
 await page.evaluate(()=>history.replaceState(null,'','/abyss/rift'));await page.reload();await expect(page.locator('#rift-start')).toBeEnabled();expect((await saved()).room_objective.carrying).toBe(true);await page.locator('#rift-auto').uncheck();await listen();
 await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();
 await page.keyboard.down('KeyD');try{await expect.poll(async()=>(await saved()).player.x,{intervals:[20],timeout:12000}).toBeGreaterThan(1440);}finally{await page.keyboard.up('KeyD');}
 await page.keyboard.down('KeyS');try{await expect.poll(async()=>(await saved()).room_objective.complete,{intervals:[20],timeout:8000}).toBe(true);}finally{await page.keyboard.up('KeyS');}
 await expect(progress).toContainText('Delivered');await expect.poll(async()=>(await saved()).status).toBe('cleared');expect((await saved()).room_objective.carrying).toBe(false);await expect.poll(()=>page.evaluate(()=>window.relicCues.filter(k=>k==='relic_delivered').length)).toBe(1);await page.locator('#rift-viewport').screenshot({path:'test-results/relic-delivered.png'});
 await page.locator('#rift-next').click();await expect.poll(async()=>(await saved()).room).toBe(2);await expect(page.locator('#rift-room-objective')).toBeHidden();
});
test('relic protocol rejects impossible carrying states and mobile HUD fits',async({page})=>{
 await page.goto('/abyss/rift?scenario=relic');await expect(page.locator('#rift-start')).toBeEnabled();const data=await(await page.request.get('/api/abyss/rift')).json();
 expect(await page.evaluate(data=>{const valid=v=>{try{window.RiftProtocol.validate(v,'GET');return true;}catch(_){return false;}};const carry=structuredClone(data);carry.run.room_objective.carrying=true;const complete=structuredClone(data);complete.run.room_objective.complete=true;const zone=structuredClone(data);zone.run.room_objective.zone.radius_x=0;return [data,carry,complete,zone].map(valid);},data)).toEqual([true,false,false,false]);
 await page.setViewportSize({width:390,height:844});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
});
