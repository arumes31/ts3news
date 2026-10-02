const {test,expect}=require('@playwright/test');
for(const reduced of [false,true])test(`recovering floor states and keyboard bypass reduced=${reduced}`,async({page},info)=>{
 test.setTimeout(90000);const errors=[];page.on('pageerror',e=>errors.push(e.message));
 await page.emulateMedia({reducedMotion:reduced?'reduce':'no-preference'});
 await page.goto('/abyss/rift?scenario=waves&condition=floor');await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('#rift-auto').uncheck();
 const read=async()=>(await(await page.request.get('/api/abyss/rift')).json()).run;
 const initial=await read();expect(initial.room_objective.floor_segments).toHaveLength(2);
 await page.locator('#rift-viewport').screenshot({path:info.outputPath('floor-warning.png'),style:'#rift-overlay{visibility:hidden!important}'});
 await page.evaluate(()=>{const play=RiftAudio.play;window.floorCues=[];RiftAudio.play=function(kind,...args){const result=play.call(this,kind,...args);if(kind.startsWith('floor_'))floorCues.push({kind,result});return result;};});
 await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();
 await expect.poll(async()=>(await read()).room_objective.floor_segments.every(p=>p.collapsed),{timeout:10000}).toBe(true);
 await expect(page.locator('#rift-minimap [data-kind="floor-gap"]')).toHaveCount(2);
 await expect.poll(()=>page.evaluate(()=>floorCues.filter(c=>c.kind==='floor_collapse'&&c.result===true).length)).toBe(2);
 await expect(page.locator('#rift-room-objective-directions')).toContainText('Floor gaps');
 await page.keyboard.down('Space');await page.keyboard.down('d');await page.waitForTimeout(450);await page.keyboard.up('d');await page.keyboard.up('Space');
 expect((await read()).player.x).toBeLessThanOrEqual(430);
 await page.keyboard.press('Escape');await expect(page.locator('#rift-paused-badge')).toHaveText('Paused');
 const paused=await read();await page.locator('#rift-viewport').screenshot({path:info.outputPath('floor-gap.png'),style:'#rift-overlay{visibility:hidden!important}'});
 await page.evaluate(()=>history.replaceState(null,'','/abyss/rift'));await page.reload();await expect(page.locator('#rift-start')).toHaveText('Resume expedition');expect((await read()).room_objective.floor_segments).toEqual(paused.room_objective.floor_segments);
 const response=await(await page.request.get('/api/abyss/rift')).json();
 expect(await page.evaluate(data=>{const bad=structuredClone(data);bad.run.room_objective.floor_segments[0].x++;try{RiftProtocol.validate(bad,'GET');return false;}catch{return true;}},response)).toBe(true);
 await page.evaluate(()=>{const play=RiftAudio.play;window.floorCues=[];RiftAudio.play=function(kind,...args){const result=play.call(this,kind,...args);if(kind.startsWith('floor_'))floorCues.push({kind,result});return result;};});
 await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();
 // Confirm each keyboard step so a slow poll cannot leave movement held long
 // enough to overshoot the narrow lane between the barricade and the gate.
 async function move(key,axis,target,greater){
  let value=(await read()).player[axis];
  while(greater?value<=target:value>=target){
   const previous=value;await page.keyboard.press(key);
   await expect.poll(async()=>{value=(await read()).player[axis];return greater?value>previous:value<previous;},{intervals:[30]}).toBe(true);
  }
 }
 // The barricade ends at x=734; clear the player's 10px footprint while
 // staying left of the enemies at x=780, facing right for the attack.
 await move('w','y',340,false);await move('d','x',750,true);await move('s','y',390,true);
 await page.keyboard.down('j');try{await expect.poll(async()=>(await read()).room_objective.next_wave_seconds).toBeGreaterThan(0);}finally{await page.keyboard.up('j');}
 await page.keyboard.press('Escape');await expect(page.locator('#rift-paused-badge')).toHaveText('Paused');
 await expect.poll(()=>page.evaluate(()=>floorCues.filter(c=>c.kind==='floor_restore'&&c.result===true).length)).toBe(2);
 const restored=await read();expect(restored.room_objective.floor_segments.every(p=>!p.collapsed&&p.collapse_in===0)).toBe(true);
 await expect(page.locator('#rift-room-objective-directions')).toContainText('Floor panels rebuilt');
 await page.setViewportSize({width:390,height:844});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
 await page.locator('#rift-viewport').screenshot({path:info.outputPath('floor-restored-mobile.png'),style:'#rift-overlay{visibility:hidden!important}'});
 expect(errors).toEqual([]);
});
test('wave-floor preview explains occupancy and recovery',async({page})=>{
 await page.goto('/abyss/rift?mission=5');await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('#rift-mission-preview > summary').click();
 await expect(page.locator('#rift-room-previews [data-terrain="floor-panel"]')).toHaveCount(2);await expect(page.locator('#rift-room-previews')).toContainText('panels rebuild between waves');
});

test('occupied panel warns until the player leaves its footprint',async({page},info)=>{
 await page.goto('/abyss/rift?scenario=waves&condition=floor');await expect(page.locator('#rift-start')).toBeEnabled();
 const read=async()=>(await(await page.request.get('/api/abyss/rift')).json()).run;
 await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();
 await page.keyboard.down('d');try{await expect.poll(async()=>(await read()).player.x,{intervals:[20]}).toBeGreaterThan(480);}finally{await page.keyboard.up('d');}
 await expect.poll(async()=>(await read()).room_objective.floor_segments[0].collapse_in).toBe(0);
 expect((await read()).room_objective.floor_segments[0].collapsed).toBe(false);
 await expect(page.locator('#rift-room-objective-directions')).toContainText('Floor waiting for occupants to leave');
 await page.locator('#rift-viewport').screenshot({path:info.outputPath('floor-occupied.png')});
 await page.keyboard.down('w');try{await expect.poll(async()=>(await read()).player.y,{intervals:[20]}).toBeLessThan(360);}finally{await page.keyboard.up('w');}
 await expect.poll(async()=>(await read()).room_objective.floor_segments[0].collapsed).toBe(true);
});
