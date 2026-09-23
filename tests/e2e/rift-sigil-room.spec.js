const {test,expect}=require('@playwright/test');
test('collect three visible sigils with movement, save midway and clear the tier',async({page})=>{
 test.setTimeout(90000);
 await page.goto('/abyss/rift?scenario=sigils');await expect(page.locator('#rift-start')).toBeEnabled();
 await expect(page.locator('#rift-room-objective-progress')).toContainText('0 / 3');await expect(page.locator('#rift-room-objective-directions')).toContainText('Sigil 3');
 await page.locator('#rift-auto').uncheck();
 const saved=async()=>(await(await page.request.get('/api/abyss/rift')).json()).run;
 expect((await saved()).status).toBe('fighting');
 await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();
 await page.locator('#rift-viewport').screenshot({path:'test-results/sigil-room.png'});
 await page.evaluate(()=>{window.sigilCues=[];const play=window.RiftAudio.play;window.RiftAudio.play=function(kind,...args){if(kind==='sigil_pickup')window.sigilCues.push(kind);return play.call(this,kind,...args);};});
 async function move(key,axis,target,increasing){await page.keyboard.down(key);try{await expect.poll(async()=>{const value=(await saved()).player[axis];return increasing?value>=target:value<=target;},{intervals:[50],timeout:15000}).toBe(true);}finally{await page.keyboard.up(key);}}
 await move('KeyD','x',435,true);await expect.poll(async()=>(await saved()).room_objective.collected).toBe(1);await expect.poll(()=>page.evaluate(()=>window.sigilCues.length)).toBe(1);
 await page.keyboard.press('Escape');await expect(page.locator('#rift-paused-badge')).toHaveText('Paused');await page.evaluate(()=>history.replaceState(null,'','/abyss/rift'));await page.reload();await expect(page.locator('#rift-start')).toBeEnabled();await expect(page.locator('#rift-room-objective-progress')).toContainText('1 / 3');
 await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();
 await move('KeyD','x',700,true);await move('KeyS','y',480,true);await move('KeyD','x',820,true);await expect.poll(async()=>(await saved()).room_objective.collected).toBe(2);
 await move('KeyD','x',1300,true);await move('KeyW','y',330,false);await move('KeyA','x',1240,false);
 await expect.poll(async()=>(await saved()).status).toBe('cleared');await expect(page.locator('#rift-room-objective-progress')).toContainText('3 / 3 · Gathered');await expect(page.locator('#rift-room-objective-help')).toContainText('Tier secured');
 await page.locator('#rift-next').click();await expect.poll(async()=>(await saved()).room).toBe(2);await expect(page.locator('#rift-room-objective')).toBeHidden();
});

test('sigil snapshots reject inconsistent or duplicate pickup progress',async({page})=>{
 await page.goto('/abyss/rift?scenario=sigils');await expect(page.locator('#rift-start')).toBeEnabled();
 const data=await(await page.request.get('/api/abyss/rift')).json();
 const results=await page.evaluate(data=>{
  const valid=value=>{try{window.RiftProtocol.validate(value,'GET');return true;}catch(_){return false;}};
  const duplicate=structuredClone(data);duplicate.run.room_objective.pickups[1].id=1;
  const count=structuredClone(data);count.run.room_objective.collected=2;
  const complete=structuredClone(data);complete.run.room_objective.complete=true;
  return [valid(data),valid(duplicate),valid(count),valid(complete)];
 },data);expect(results).toEqual([true,false,false,false]);
 await page.setViewportSize({width:390,height:844});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);await expect(page.locator('#rift-room-objective-progress')).toContainText('0 / 3');
});
