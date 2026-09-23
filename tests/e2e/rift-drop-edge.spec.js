const {test,expect}=require('@playwright/test');
for(const reduced of [false,true])test((reduced?'reduced motion: ':'')+'one-way ledge drops safely, persists and allows a return around its end',async({page})=>{
 await page.emulateMedia({reducedMotion:reduced?'reduce':'no-preference'});
 await page.addInitScript(()=>{window.dropCues=0;window.dropOscillators=0;let nodes=0;const create=BaseAudioContext.prototype.createOscillator;BaseAudioContext.prototype.createOscillator=function(...args){nodes++;return create.apply(this,args);};const timer=setInterval(()=>{if(!window.RiftAudio)return;clearInterval(timer);const play=RiftAudio.play;RiftAudio.play=function(kind,...args){const before=nodes,result=play.call(this,kind,...args);if(kind==='ledge_drop'){window.dropCues++;window.dropOscillators+=nodes-before;}return result;};},10);});
 await page.goto('/abyss/rift?scenario=drop-edge');await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('#rift-auto').uncheck();
 const saved=async()=>(await(await page.request.get('/api/abyss/rift')).json()).run;
 await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();
 await expect(page.locator('#rift-terrain-hint')).toHaveAttribute('data-kind','ledge');await expect(page.locator('#rift-terrain-hint')).toContainText('Return around either end');
 const hp=(await saved()).player.hp;
 await page.locator('#rift-viewport').screenshot({path:'test-results/drop-edge-upper'+(reduced?'-reduced':'')+'.png'});
 await page.keyboard.down('KeyS');try{await expect.poll(async()=>(await saved()).player.y,{intervals:[20]}).toBeGreaterThanOrEqual(425);}finally{await page.keyboard.up('KeyS');}
 await page.keyboard.down('KeyW');await page.waitForTimeout(450);await page.keyboard.up('KeyW');
 let run=await saved();expect(run.player.y).toBeGreaterThanOrEqual(425);expect(run.player.hp).toBe(hp);expect(run.stats.jumps||0).toBe(0);expect(run.stats.kills||0).toBe(0);expect(run.drops).toHaveLength(0);
 await expect.poll(()=>page.evaluate(()=>window.dropCues)).toBe(1);expect(await page.evaluate(()=>window.dropOscillators)).toBeGreaterThan(0);
 await page.locator('#rift-viewport').screenshot({path:'test-results/drop-edge-lower'+(reduced?'-reduced':'')+'.png'});
 await page.evaluate(()=>history.replaceState(null,'','/abyss/rift'));await page.reload();await expect(page.locator('#rift-start')).toBeEnabled();expect((await saved()).level.rooms[0].drop_edges[0].id).toBe('ledge-workshop');expect((await saved()).player.y).toBeGreaterThanOrEqual(425);
 await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();
 await page.keyboard.down('KeyA');try{await expect.poll(async()=>(await saved()).player.x,{intervals:[20]}).toBeLessThan(240);}finally{await page.keyboard.up('KeyA');}
 await page.keyboard.down('KeyW');try{await expect.poll(async()=>(await saved()).player.y,{intervals:[20]}).toBeLessThan(350);}finally{await page.keyboard.up('KeyW');}
 await page.setViewportSize({width:390,height:844});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
});

test('ledge protocol rejects impossible geometry and duplicate IDs',async({page})=>{
 await page.goto('/abyss/rift?scenario=drop-edge');await expect(page.locator('#rift-start')).toBeEnabled();const data=await(await page.request.get('/api/abyss/rift')).json();
 expect(await page.evaluate(data=>{const valid=v=>{try{RiftProtocol.validate(v,'GET');return true;}catch(_){return false;}};const changed=fn=>{const v=structuredClone(data);fn(v.run.level.rooms[0].drop_edges);return v;};return [data,changed(e=>e[0].landing_y=365),changed(e=>e[0].landing_y=510),changed(e=>e[0].w=-1),changed(e=>e[0].x=1550),changed(e=>e.push({...e[0]}))].map(valid);},data)).toEqual([true,false,false,false,false,false]);
});
test('campaign preview describes the one-way return path',async({page})=>{
 await page.goto('/abyss/rift?mission=1');await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('#rift-mission-preview > summary').click();await expect(page.locator('#rift-room-previews [data-terrain="ledge"]')).toHaveCount(1);await expect(page.locator('#rift-room-previews [data-terrain="ledge"] title')).toContainText('return around either end');
});

test('an enemy below the ledge walks around an end to reach the upper floor',async({page})=>{
 await page.goto('/abyss/rift?scenario=drop-edge-pursuit');await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('#rift-auto').uncheck();await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();
 const saved=async()=>(await(await page.request.get('/api/abyss/rift')).json()).run;
 await expect.poll(async()=>(await saved()).enemies[0].y,{timeout:10000,intervals:[40]}).toBeLessThan(355);
 expect((await saved()).enemies[0].hp).toBe(100);expect((await saved()).stats.kills||0).toBe(0);
});
