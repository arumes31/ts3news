const {test,expect}=require('@playwright/test');
test('contest, charge, leave, reload and finish a circle room',async({page})=>{
 test.setTimeout(90000);
 await page.goto('/abyss/rift?scenario=circle');await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('#rift-auto').uncheck();
 const progress=page.locator('#rift-room-objective-progress');await expect(progress).toContainText('Hold the circle');
 const saved=async()=>(await(await page.request.get('/api/abyss/rift')).json()).run;
 const listen=()=>page.evaluate(()=>{window.circleCues=[];const play=window.RiftAudio.play;window.RiftAudio.play=function(kind,...args){if(kind.startsWith('circle_'))window.circleCues.push(kind);return play.call(this,kind,...args);};});
 await listen();await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();await expect(progress).toContainText('Contested');expect((await saved()).room_objective.seconds).toBe(0);await page.locator('#rift-viewport').screenshot({path:'test-results/circle-room.png'});
 await page.keyboard.press('KeyJ');await expect(progress).toContainText('Charging');await expect.poll(async()=>(await saved()).room_objective.seconds).toBeGreaterThan(1);
 await page.keyboard.down('KeyA');try{await expect.poll(async()=>(await saved()).player.x).toBeLessThan(390);}finally{await page.keyboard.up('KeyA');}
 await expect(progress).toContainText('Move onto the circle');await page.keyboard.press('Escape');await expect(page.locator('#rift-paused-badge')).toHaveText('Paused');const earned=(await saved()).room_objective.seconds;
 await page.evaluate(()=>history.replaceState(null,'','/abyss/rift'));await page.reload();await expect(page.locator('#rift-start')).toBeEnabled();expect((await saved()).room_objective.seconds).toBe(earned);await page.locator('#rift-auto').uncheck();await listen();
 await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();await page.keyboard.down('KeyD');try{await expect.poll(async()=>(await saved()).player.x).toBeGreaterThan(440);}finally{await page.keyboard.up('KeyD');}
 await expect.poll(async()=>(await saved()).status,{timeout:25000}).toBe('cleared');await expect(progress).toContainText('15 / 15 s · Charged');await expect.poll(()=>page.evaluate(()=>window.circleCues.filter(k=>k==='circle_complete').length)).toBe(1);
 await page.locator('#rift-next').click();await expect.poll(async()=>(await saved()).room).toBe(2);await expect(page.locator('#rift-room-objective')).toBeHidden();
});
test('circle protocol rejects impossible charge and mobile HUD fits',async({page})=>{
 await page.goto('/abyss/rift?scenario=circle');await expect(page.locator('#rift-start')).toBeEnabled();const data=await(await page.request.get('/api/abyss/rift')).json();
 expect(await page.evaluate(data=>{const valid=value=>{try{window.RiftProtocol.validate(value,'GET');return true;}catch(_){return false;}};const over=structuredClone(data);over.run.room_objective.seconds=16;const complete=structuredClone(data);complete.run.room_objective.complete=true;const zone=structuredClone(data);zone.run.room_objective.zone.radius_x=0;return [valid(data),valid(over),valid(complete),valid(zone)];},data)).toEqual([true,false,false,false]);
 await page.setViewportSize({width:390,height:844});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
});

test('ended expedition cannot display an active circle charge',async({page})=>{
 await page.route('**/api/abyss/rift',async route=>{
  const response=await route.fetch();const data=await response.json();
  if(route.request().method()==='GET'&&data.run?.room_objective?.kind==='hold_circle'){data.run.status='defeated';data.run.player.hp=0;data.run.room_objective.charging=true;await route.fulfill({response,json:data});}else await route.fulfill({response});
 });
 await page.goto('/abyss/rift?scenario=circle');await expect(page.locator('#rift-start')).toBeEnabled();await expect(page.locator('#rift-room-objective-progress')).toContainText('Expedition ended');await expect(page.locator('#rift-room-objective-help')).toContainText('not secured');
});
