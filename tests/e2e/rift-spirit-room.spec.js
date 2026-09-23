const {test,expect}=require('@playwright/test');
test('clear threats, leave, reload and escort the spirit to safety',async({page})=>{
 test.setTimeout(90000);
 await page.goto('/abyss/rift?scenario=spirit');await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('#rift-auto').uncheck();
 const saved=async()=>(await(await page.request.get('/api/abyss/rift')).json()).run;
 const progress=page.locator('#rift-room-objective-progress');
 await page.evaluate(()=>{window.initialSpiritCues=[];const play=window.RiftAudio.play;window.RiftAudio.play=function(kind,...args){if(kind.startsWith('spirit_'))window.initialSpiritCues.push(kind);return play.call(this,kind,...args);};});
 await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();await expect(progress).toContainText('Clear nearby enemies');expect((await saved()).room_objective.escort.x).toBe(350);await expect.poll(()=>page.evaluate(()=>window.initialSpiritCues.filter(k=>k==='spirit_threat').length)).toBe(1);await page.locator('#rift-viewport').screenshot({path:'test-results/spirit-threat.png'});
 await page.keyboard.press('KeyJ');await expect(progress).toContainText('Escorting');await expect.poll(()=>page.evaluate(()=>window.initialSpiritCues.filter(k=>k==='spirit_move').length)).toBe(1);await expect.poll(async()=>(await saved()).room_objective.escort.x).toBeGreaterThan(385);
 await page.keyboard.down('KeyA');try{await expect.poll(async()=>(await saved()).player.x,{intervals:[20]}).toBeLessThan(230);}finally{await page.keyboard.up('KeyA');}await expect(progress).toContainText('Stay near the spirit');
 await page.keyboard.press('Escape');await expect(page.locator('#rift-paused-badge')).toHaveText('Paused');const position=(await saved()).room_objective.escort.x;
 await page.evaluate(()=>history.replaceState(null,'','/abyss/rift'));await page.reload();await expect(page.locator('#rift-start')).toBeEnabled();expect((await saved()).room_objective.escort.x).toBe(position);await page.locator('#rift-auto').uncheck();
 await page.evaluate(()=>{window.spiritCues=[];const play=window.RiftAudio.play;window.RiftAudio.play=function(kind,...args){if(kind.startsWith('spirit_'))window.spiritCues.push(kind);return play.call(this,kind,...args);};});
 await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();const held=new Set();let screenshot=false;
 try{for(let n=0;n<700;n++){
  const run=await saved(),o=run.room_objective;if(o.complete)break;
  if(!screenshot&&o.escort_moving&&o.escort.x>600){await page.locator('#rift-viewport').screenshot({path:'test-results/spirit-escort.png'});screenshot=true;}
  const dx=o.escort.x-50-run.player.x,dy=o.escort.y-run.player.y,wanted=new Set();if(Math.abs(dx)>12)wanted.add(dx>0?'KeyD':'KeyA');if(Math.abs(dy)>12)wanted.add(dy>0?'KeyS':'KeyW');
  for(const key of held)if(!wanted.has(key)){await page.keyboard.up(key);held.delete(key);}for(const key of wanted)if(!held.has(key)){await page.keyboard.down(key);held.add(key);}await page.waitForTimeout(60);
 }}finally{for(const key of held)await page.keyboard.up(key);}
 await expect.poll(async()=>(await saved()).status).toBe('cleared');await expect(progress).toContainText('Spirit 100% · Safe at the exit');const result=await saved();expect(result.stats.kills).toBe(1);expect(result.drops.length).toBe(1);await expect.poll(()=>page.evaluate(()=>window.spiritCues.filter(k=>k==='spirit_arrived').length)).toBe(1);
 await page.locator('#rift-next').click();await expect.poll(async()=>(await saved()).room).toBe(2);await expect(page.locator('#rift-room-objective')).toBeHidden();
});
test('escort snapshots reject impossible states and mobile HUD fits',async({page})=>{
 await page.goto('/abyss/rift?scenario=spirit');await expect(page.locator('#rift-start')).toBeEnabled();const data=await(await page.request.get('/api/abyss/rift')).json();
 expect(await page.evaluate(data=>{const valid=v=>{try{window.RiftProtocol.validate(v,'GET');return true;}catch(_){return false;}};const position=structuredClone(data);position.run.room_objective.escort.x=1500;const speed=structuredClone(data);speed.run.room_objective.escort.speed=300;const complete=structuredClone(data);complete.run.room_objective.complete=true;const threat=structuredClone(data);threat.run.room_objective.escort_moving=true;threat.run.room_objective.contested=true;return [data,position,speed,complete,threat].map(valid);},data)).toEqual([true,false,false,false,false]);
 await page.setViewportSize({width:390,height:844});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
});
