const {test,expect}=require('@playwright/test');
test('wave gate closes, persists, blocks a jump and leaves a walkable bypass',async({page},info)=>{
 test.setTimeout(90000);const errors=[];page.on('pageerror',e=>errors.push(e.message));
 await page.goto('/abyss/rift?scenario=waves&condition=gate');await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('#rift-auto').uncheck();
 const saved=async()=>(await(await page.request.get('/api/abyss/rift')).json()).run;
 await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();
 await expect.poll(async()=>(await saved()).room_objective.gate.closed).toBe(true);
 await expect(page.locator('#rift-room-objective-directions')).toContainText('Gate closed');
 await page.keyboard.press('Escape');await expect(page.locator('#rift-paused-badge')).toHaveText('Paused');const before=(await saved()).room_objective.gate;
 await page.evaluate(()=>history.replaceState(null,'','/abyss/rift'));await page.reload();await expect(page.locator('#rift-start')).toBeEnabled();expect((await saved()).room_objective.gate).toEqual(before);await page.locator('#rift-auto').uncheck();
 await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();
 await page.keyboard.down('Space');await page.keyboard.down('KeyD');await page.waitForTimeout(750);await page.keyboard.up('KeyD');await page.keyboard.up('Space');
 expect((await saved()).player.x).toBeLessThanOrEqual(before.x-10);
 await page.locator('#rift-canvas').screenshot({path:info.outputPath('wave-gate-closed.png')});
 const move=async(key,axis,target,greater)=>{await page.keyboard.down(key);try{await expect.poll(async()=>{const v=(await saved()).player[axis];return greater?v>target:v<target;},{timeout:12000,intervals:[30]}).toBe(true);}finally{await page.keyboard.up(key);}};
 await move('KeyW','y',340,false);await move('KeyD','x',1150,true);await move('KeyS','y',383,true);
 await page.keyboard.down('KeyJ');try{await expect.poll(async()=>(await saved()).room_objective.next_wave_seconds).toBeGreaterThan(0);}finally{await page.keyboard.up('KeyJ');}
 await page.keyboard.press('Escape');await expect(page.locator('#rift-paused-badge')).toHaveText('Paused');
 const between=await saved();expect(between.room_objective.gate.closed).toBe(false);expect(between.room_objective.gate.close_in).toBe(0);expect(between.events.some(e=>e.kind==='wave_gate_open')).toBe(true);
 await page.evaluate(()=>{document.querySelector('#rift-overlay').hidden=true;});
 await page.locator('#rift-canvas').screenshot({path:info.outputPath('wave-gate-open.png')});expect(errors).toEqual([]);
});

test('wave gate protocol rejects contradictory state and altered frozen geometry',async({page})=>{
 await page.goto('/abyss/rift?scenario=waves');await expect(page.locator('#rift-start')).toBeEnabled();const data=await(await page.request.get('/api/abyss/rift')).json();
 const result=await page.evaluate(data=>{
  const valid=v=>{try{RiftProtocol.validate(v,'GET');return true;}catch(_){return false;}};
  const altered=structuredClone(data);altered.run.room_objective.gate.x++;
  const closed=structuredClone(data);closed.run.room_objective.gate.closed=true;
  const timer=structuredClone(data);timer.run.room_objective.gate.close_in=2;
  const missing=structuredClone(data);delete missing.run.room_objective.gate;
  const legacy=structuredClone(data);delete legacy.run.room_objective.gate;delete legacy.run.level.rooms[legacy.run.room].wave_gate;
  return [data,altered,closed,timer,missing,legacy].map(valid);
 },data);expect(result).toEqual([true,false,false,false,false,true]);
});
