const {test,expect}=require('@playwright/test');
test('follow moving beacons through cover and retain capture on reload',async({page})=>{
 test.setTimeout(90000);
 await page.goto('/abyss/rift?scenario=beacons');await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('#rift-auto').uncheck();
 const saved=async()=>(await(await page.request.get('/api/abyss/rift')).json()).run;
 const progress=page.locator('#rift-room-objective-progress');await expect(progress).toContainText('Beacons 0/3');const startX=(await saved()).room_objective.zone.x;
 await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();await expect.poll(async()=>(await saved()).room_objective.seconds).toBeGreaterThan(.4);
 await page.keyboard.press('Escape');await expect(page.locator('#rift-paused-badge')).toHaveText('Paused');const partial=(await saved()).room_objective;expect(partial.zone.x).not.toBe(startX);expect(partial.collected).toBe(0);await page.locator('#rift-viewport').screenshot({path:'test-results/beacon-capture.png'});
 await page.evaluate(()=>history.replaceState(null,'','/abyss/rift'));await page.reload();await expect(page.locator('#rift-start')).toBeEnabled();const restored=(await saved()).room_objective;expect(restored.seconds).toBe(partial.seconds);expect(restored.beacon_time).toBe(partial.beacon_time);expect(restored.zone).toEqual(partial.zone);await page.locator('#rift-auto').uncheck();
 await page.evaluate(()=>{window.beaconCues=[];const play=window.RiftAudio.play;window.RiftAudio.play=function(kind,...args){if(kind.startsWith('beacon'))window.beaconCues.push(kind);return play.call(this,kind,...args);};});
 await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();
 const held=new Set();
 try{for(let n=0;n<900;n++){
  const run=await saved(),o=run.room_objective;if(o.complete)break;
  let x=o.zone.x,y=o.zone.y;
  if(o.collected===1&&run.player.y<460){x=600;y=run.player.y;if(Math.abs(run.player.x-600)<15)y=480;}
  if(o.collected===2&&run.player.x<1100){x=1120;y=480;}
  const dx=x-run.player.x,dy=y-run.player.y,wanted=new Set();if(Math.abs(dx)>12)wanted.add(dx>0?'KeyD':'KeyA');if(Math.abs(dy)>12)wanted.add(dy>0?'KeyS':'KeyW');
  for(const key of held)if(!wanted.has(key)){await page.keyboard.up(key);held.delete(key);}for(const key of wanted)if(!held.has(key)){await page.keyboard.down(key);held.add(key);}await page.waitForTimeout(60);
 }}finally{for(const key of held)await page.keyboard.up(key);}
 await expect.poll(async()=>(await saved()).status).toBe('cleared');await expect(progress).toContainText('Beacons 3/3 · Captured');expect(await page.evaluate(()=>window.beaconCues.filter(k=>k==='beacon_captured').length)).toBe(3);expect(await page.evaluate(()=>window.beaconCues.filter(k=>k==='beacons_complete').length)).toBe(1);
 await page.locator('#rift-next').click();await expect.poll(async()=>(await saved()).room).toBe(2);await expect(page.locator('#rift-room-objective')).toBeHidden();
});
test('beacon snapshots validate motion and charge; mobile HUD fits',async({page})=>{
 await page.goto('/abyss/rift?scenario=beacons');await expect(page.locator('#rift-start')).toBeEnabled();const data=await(await page.request.get('/api/abyss/rift')).json();
 expect(await page.evaluate(data=>{const valid=v=>{try{window.RiftProtocol.validate(v,'GET');return true;}catch(_){return false;}};const position=structuredClone(data);position.run.room_objective.zone.x+=10;const charge=structuredClone(data);charge.run.room_objective.seconds=4;const complete=structuredClone(data);complete.run.room_objective.complete=true;return [data,position,charge,complete].map(valid);},data)).toEqual([true,false,false,false]);
 await page.setViewportSize({width:390,height:844});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
});
